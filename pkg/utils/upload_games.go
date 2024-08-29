package utils

import (
	"context"
	"encoding/csv"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"google.golang.org/api/option"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/PixelzOrg/PHOLE.git/pkg/database"
	"github.com/PixelzOrg/PHOLE.git/pkg/models"
	fa "github.com/PixelzOrg/PHOLE.git/pkg/utils/firebase"
	"github.com/algolia/algoliasearch-client-go/v3/algolia/search"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

func setupAlgoliaClient() (*search.Client, error) {
	client := search.NewClient("9QQLSH3XI9", "d93d48a7153add36381df4649781a377")

	_, err := client.ListIndices()
	if err != nil {
		return nil, fmt.Errorf("Algolia permissions error: %v", err)
	}

	return client, nil
}

func MigrateFromFirestore(ctx context.Context, fa *fa.FirebaseApp, h database.Handler, algolia *search.Client, s3BucketName string) error {
	algoliaClient, err := setupAlgoliaClient()
	if err != nil {
		return fmt.Errorf("failed to setup Algolia client: %v", err)
	}

	index := algoliaClient.InitIndex("prod_GAMES")

	_, err = index.GetSettings()
	if err != nil {
		return fmt.Errorf("Algolia index permissions error: %v", err)
	}

	log.Printf("Index, %v", index)

	_, uploader, err := setupAWSSession()
	if err != nil {
		log.Fatal(err)
		return err
	}

	storageClient, err := storage.NewClient(ctx, option.WithCredentialsFile("./firebase.json"))
	if err != nil {
		log.Fatal(err)
		return err
	}

	genres := make(map[string]*models.Genre)

	iter := fa.Firestore.Collection("games").Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("Failed to iterate over documents: %v", err)
			continue
		}

		data := doc.Data()

		getStringDefault := func(key string) string {
			if v, ok := data[key].(string); ok {
				return v
			}
			return ""
		}

		getIntDefault := func(key string) int {
			if v, ok := data[key].(int64); ok {
				return int(v)
			}
			return 0
		}

		getBoolDefault := func(key string) bool {
			if v, ok := data[key].(bool); ok {
				return v
			}
			return false
		}

		thumbnailPath := getStringDefault("thumbnailFileName")
		var s3Key string
		if thumbnailPath != "" {
			s3Key, err = uploadThumbnailToS3(ctx, storageClient, uploader, thumbnailPath, s3BucketName)
			if err != nil {
				log.Printf("Failed to upload thumbnail to S3: %v", err)
			}
		}

		genreName := getStringDefault("genre")
		var genre *models.Genre
		if genreName != "" {
			if existingGenre, ok := genres[genreName]; ok {
				genre = existingGenre
			} else {
				genre = &models.Genre{Name: genreName}
				if err := h.DB.FirstOrCreate(genre, models.Genre{Name: genreName}).Error; err != nil {
					log.Printf("Failed to create genre: %v", err)
					genre = nil
				} else {
					genres[genreName] = genre
				}
			}
		}

		game := models.Game{
			Title:             getStringDefault("title"),
			Description:       getStringDefault("description"),
			PlayCount:         getIntDefault("views"),
			LikeCount:         getIntDefault("likeCount"),
			CommentCount:      0,
			BookmarkCount:     0,
			IsFeatured:        false,
			ButtonMapping:     getBoolDefault("buttonMapping"),
			EmbedLink:         getStringDefault("embedLink"),
			GameType:          getStringDefault("gameType"),
			ThumbnailFileName: s3Key,
			IsLandscape:       !getBoolDefault("isTiles"),
			IsClaimed:         false,
		}

		if genre != nil {
			game.GenreID = genre.ID
		}

		if err := h.DB.Create(&game).Error; err != nil {
			log.Printf("Failed to insert game into database: %v", err)
			continue
		}

		log.Printf("Raw tags string: %s", getStringDefault("tags"))
		tagNames := strings.Split(getStringDefault("tags"), ",")
		log.Printf("Split tag names: %v", tagNames)

		for _, tagName := range tagNames {
			tagName = strings.TrimSpace(tagName)
			if tagName == "" {
				continue
			}

			log.Printf("Processing tag: %s", tagName)

			var tag models.Tag
			if err := h.DB.Where(models.Tag{Name: tagName}).FirstOrCreate(&tag).Error; err != nil {
				log.Printf("Failed to create or find tag: %v", err)
				continue
			}

			log.Printf("Tag created or found: ID=%d, Name=%s", tag.ID, tag.Name)

			if err := h.DB.Exec("INSERT INTO game_tags (game_id, tag_id) VALUES (?, ?)", game.ID, tag.ID).Error; err != nil {
				log.Printf("Failed to associate tag with game: %v", err)
			} else {
				log.Printf("Associated tag %s with game %s", tag.Name, game.ID)
			}
		}

		var gameTags []string
		err = h.DB.Raw(`
            SELECT t.name 
            FROM tags t
            JOIN game_tags gt ON t.id = gt.tag_id
            WHERE gt.game_id = ?
        `, game.ID).Pluck("name", &gameTags).Error

		if err != nil {
			log.Printf("Failed to fetch tags for game: %v", err)
		}

		algoliaObject := map[string]interface{}{
			"objectID":     game.ID,
			"title":        game.Title,
			"description":  game.Description,
			"isFeatured":   game.IsFeatured,
			"isLandscape":  game.IsLandscape,
			"thumbnailURL": game.ThumbnailFileName,
		}

		if genre != nil {
			algoliaObject["genre"] = genre.Name
			algoliaObject["genreID"] = genre.ID
		}

		if len(gameTags) > 0 {
			algoliaObject["tags"] = strings.Join(gameTags, ", ")
		}

		_, err = index.SaveObject(algoliaObject)
		if err != nil {
			log.Printf("Failed to index game in Algolia: %v", err)
		}

		log.Printf("Inserted and indexed game: %v\n", game.Title)
	}

	return nil
}

func uploadThumbnailToS3(ctx context.Context, storageClient *storage.Client, uploader *s3manager.Uploader, thumbnailPath, s3BucketName string) (string, error) {
	bucket := storageClient.Bucket("joystick-database.appspot.com")

	obj := bucket.Object(thumbnailPath)

	reader, err := obj.NewReader(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create reader: %v", err)
	}
	defer reader.Close()

	fileExt := filepath.Ext(thumbnailPath)
	s3Key := fmt.Sprintf("thumbnails/%s%s", uuid.New().String(), fileExt)

	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(s3BucketName),
		Key:    aws.String(s3Key),
		Body:   reader,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file to S3: %v", err)
	}

	return s3Key, nil
}

func getTagNames(tags []models.Tag) []string {
	var names []string
	for _, tag := range tags {
		if tag.Name != "" {
			names = append(names, tag.Name)
		}
	}
	return names
}

func setupAWSSession() (*session.Session, *s3manager.Uploader, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
		Credentials: credentials.NewStaticCredentials(
			"AKIAQMZWSSNHZPYWS7VB",
			"fqZlg40Lwn3jeQGKi6pByMBmUtk9yrPEfVA9Qh6j",
			""),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create AWS session: %v", err)
	}
	uploader := s3manager.NewUploader(sess)

	return sess, uploader, nil
}

func MigrateFromCSV(ctx context.Context, h database.Handler, algolia *search.Client, s3BucketName, csvFilePath string) error {
	index := algolia.InitIndex("prod_GAMES")

	_, uploader, err := setupAWSSession()
	if err != nil {
		return err
	}

	file, err := os.Open(csvFilePath)
	if err != nil {
		return fmt.Errorf("failed to open CSV file: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read CSV header: %v", err)
	}

	columnIndex := make(map[string]int)
	for i, column := range header {
		columnIndex[column] = i
	}

	genres := make(map[string]*models.Genre)

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Failed to read CSV row: %v", err)
			continue
		}

		title := row[columnIndex["Title"]]
		description := row[columnIndex["Description"]]
		gameLink := row[columnIndex["Game Link"]]
		thumbnailLink := extractURL(row[columnIndex["Thumbnail Link"]])
		isLandscape, _ := strconv.ParseBool(row[columnIndex["isLandscape"]])
		genreName := row[columnIndex["Genre"]]
		tags := strings.Split(row[columnIndex["Tags"]], ",")

		s3Key, err := uploadThumbnailToS3FromURL(uploader, thumbnailLink, s3BucketName)
		if err != nil {
			log.Printf("Failed to upload thumbnail to S3: %v", err)
		}

		var genre *models.Genre
		if genreName != "" {
			if existingGenre, ok := genres[genreName]; ok {
				genre = existingGenre
			} else {
				genre = &models.Genre{Name: genreName}
				if err := h.DB.FirstOrCreate(genre, models.Genre{Name: genreName}).Error; err != nil {
					log.Printf("Failed to create genre: %v", err)
					genre = nil
				} else {
					genres[genreName] = genre
				}
			}
		}

		game := models.Game{
			Title:             title,
			Description:       description,
			EmbedLink:         gameLink,
			ThumbnailFileName: s3Key,
			IsLandscape:       isLandscape,
			IsFeatured:        false,
			IsClaimed:         false,
		}

		if genre != nil {
			game.GenreID = genre.ID
		}

		if err := h.DB.Create(&game).Error; err != nil {
			log.Printf("Failed to insert game into database: %v", err)
			continue
		}

		for _, tagName := range tags {
			tagName = strings.TrimSpace(tagName)
			if tagName == "" {
				continue
			}

			var tag models.Tag
			if err := h.DB.Where(models.Tag{Name: tagName}).FirstOrCreate(&tag).Error; err != nil {
				log.Printf("Failed to create or find tag: %v", err)
				continue
			}

			if err := h.DB.Exec("INSERT INTO game_tags (game_id, tag_id) VALUES (?, ?)", game.ID, tag.ID).Error; err != nil {
				log.Printf("Failed to associate tag with game: %v", err)
			}
		}

		algoliaObject := map[string]interface{}{
			"objectID":     game.ID,
			"title":        game.Title,
			"description":  game.Description,
			"isFeatured":   game.IsFeatured,
			"isLandscape":  game.IsLandscape,
			"thumbnailURL": game.ThumbnailFileName,
			"tags":         strings.Join(tags, ", "),
		}

		if genre != nil {
			algoliaObject["genre"] = genre.Name
			algoliaObject["genreID"] = genre.ID
		}

		_, err = index.SaveObject(algoliaObject)
		if err != nil {
			log.Printf("Failed to index game in Algolia: %v", err)
		}

		log.Printf("Inserted and indexed game: %v\n", game.Title)
	}

	return nil
}

func extractURL(s string) string {
	start := strings.Index(s, "(")
	end := strings.Index(s, ")")
	if start != -1 && end != -1 && start < end {
		return s[start+1 : end]
	}
	return s
}

func uploadThumbnailToS3FromURL(uploader *s3manager.Uploader, url, s3BucketName string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download thumbnail: %v", err)
	}
	defer resp.Body.Close()

	fileExt := ".jpg"
	s3Key := fmt.Sprintf("thumbnails/%s%s", uuid.New().String(), fileExt)
	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(s3BucketName),
		Key:    aws.String(s3Key),
		Body:   resp.Body,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file to S3: %v", err)
	}

	return s3Key, nil
}
