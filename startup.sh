#!/bin/sh

## Print environment variables
#echo "Environment variables:"
#env | sort

# Download firebase.json if it doesn't exist
if [ ! -f /app/config/firebase.json ]; then
    echo "Downloading firebase.json"
    aws s3 cp s3://elasticbeanstalk-us-east-1-027494880079/firebase.json /app/config/firebase.json
fi

# Create prod.env file
echo "Creating prod.env file"
mkdir -p /app/pkg/config/envs
cat << EOF > /app/pkg/config/envs/prod.env
BACKUP_FIREBASE_PATH=${BACKUP_FIREBASE_PATH}
DBPORT=${DBPORT}
FIREBASE_PATH=${FIREBASE_PATH}
HOST=${HOST}
PASSWORD=${PASSWORD}
PORT=${PORT}
USERNAME=${USERNAME}
REDIS_PATH=${REDIS_PATH}
ALGOLIA_KEY=${ALGOLIA_KEY}
ALGOLIA_APP_ID=${ALGOLIA_APP_ID}
EOF

## Print contents of prod.env (make sure to mask sensitive data)
#echo "Contents of prod.env:"
#cat /app/pkg/config/envs/prod.env

# Start your application
echo "Starting application"
/app/main &
APP_PID=$!

# Wait for a short time to allow the app to start
sleep 10

# Check if the app is still running
if ! kill -0 $APP_PID 2>/dev/null; then
    echo "Application crashed. Printing logs:"
    docker logs $(docker ps -aq --filter ancestor=${DOCKER_IMAGE_NAME}:${DOCKER_TAG} --latest)
    exit 1
fi

# Wait for the app to finish
wait $APP_PID