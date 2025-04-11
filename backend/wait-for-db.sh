#!/bin/sh
echo "Waiting for MariaDB..."

until nc -z -v -w30 db 3306
do
  echo "Waiting for database connection..."
  sleep 2
done

echo "MariaDB is up - starting app"
exec "$@"
