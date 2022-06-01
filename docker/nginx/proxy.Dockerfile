FROM nginx:1.21.4
RUN openssl genrsa -des3 -passout file:rsapassword.txt -out madock.key 2048
{{{SSL_CREATE_BY_HOST_NAMES}}}