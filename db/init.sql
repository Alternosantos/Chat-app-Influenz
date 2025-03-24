CREATE DATABASE chat_app;
CREATE USER 'appuser'@'%' IDENTIFIED BY 'securepassword';
GRANT ALL PRIVILEGES ON chat_app.* TO 'appuser'@'%';
FLUSH PRIVILEGES;
