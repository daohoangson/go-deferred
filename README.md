# go-deferred

[![Build Status](https://travis-ci.org/daohoangson/go-deferred.svg?branch=master)](https://travis-ci.org/daohoangson/go-deferred)

[![Docker Hub](https://images.microbadger.com/badges/image/daohoangson/go-deferred.svg)](https://microbadger.com/images/daohoangson/go-deferred)

## Environment variables:

- `DEFERRED_COOLDOWN_DURATION` default=`60s`
- `DEFERRED_DUMP_RESPONSE_ON_PARSE_ERROR` default=`no`
- `DEFERRED_ERRORS_BEFORE_QUITTING` default=`3`
- `DEFERRED_HTTP_CLIENT_TIMEOUT` default=`61s`
- `DEFERRED_LOG_LEVEL` default=`info`
- `DEFERRED_MAX_HITS_PER_LOOP` default=`5`
- `DEFERMON_PORT` default=`80`
- `DEFERMON_SECRET` default=`s3cr3t`

## Docker usage

### Runner mode

Loop through deferred.php until there is nothing left:

```bash
docker run --rm daohoangson/go-deferred deferred https://xfrocks.com/deferred.php
```

It's okie to supply multiple URLs at once:

```bash
docker run --rm daohoangson/go-deferred deferred \
  https://xfrocks.com/deferred.php \
  https://tinhte.vn/deferred.php
```

### Daemon mode

Start a daemon at port 8080 with some secret. Usable with XenForo add-on [GoDeferred](https://github.com/daohoangson/GoDeferred).

```bash
docker run --rm -p 8080:8080 daohoangson/go-deferred defermon 8080 s3cr3t
```

## Heroku / Dokku deployment

Just clone this repo and push to deploy the daemon on Heroku / Dokku.
