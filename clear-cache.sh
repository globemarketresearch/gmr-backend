#!/bin/bash

# Clear Redis cache for blogs and press releases
# This is needed after updating the data structure to include author details

source .env

echo "Clearing Redis cache for blogs and press releases..."

# Use redis-cli to delete cache patterns
redis-cli -h $REDIS_HOST -p $REDIS_PORT -a $REDIS_PASSWORD --no-auth-warning --scan --pattern "blog:*" | xargs -r redis-cli -h $REDIS_HOST -p $REDIS_PORT -a $REDIS_PASSWORD --no-auth-warning del
redis-cli -h $REDIS_HOST -p $REDIS_PORT -a $REDIS_PASSWORD --no-auth-warning --scan --pattern "blogs:*" | xargs -r redis-cli -h $REDIS_HOST -p $REDIS_PORT -a $REDIS_PASSWORD --no-auth-warning del
redis-cli -h $REDIS_HOST -p $REDIS_PORT -a $REDIS_PASSWORD --no-auth-warning --scan --pattern "press_release:*" | xargs -r redis-cli -h $REDIS_HOST -p $REDIS_PORT -a $REDIS_PASSWORD --no-auth-warning del
redis-cli -h $REDIS_HOST -p $REDIS_PORT -a $REDIS_PASSWORD --no-auth-warning --scan --pattern "press_releases:*" | xargs -r redis-cli -h $REDIS_HOST -p $REDIS_PORT -a $REDIS_PASSWORD --no-auth-warning del

echo "Cache cleared successfully!"
