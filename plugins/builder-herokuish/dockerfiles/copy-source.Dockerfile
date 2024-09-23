ARG APP_IMAGE
FROM $APP_IMAGE

ARG DOKKU_APP_USER=herokuishuser
ARG TRACE
RUN TRACE=$TRACE USER=$DOKKU_APP_USER /exec true
COPY --chown=$DOKKU_APP_USER . /app
WORKDIR /app
ENV HEROKUISH_DISABLE_CHOWN true
