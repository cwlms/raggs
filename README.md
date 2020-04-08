# Data Stream Aggregator

Simple lightweight API meant to accept data from sparse data streams, cache 
that data, and make it more complete as different attributes become present.
Redis is the caching mechanism and should be tuned based on your desired cache 
eviction poicy. API accepts a post request with a JSON payload that is written 
to a pipeline and flushed at configurable intervals into Redis. Flush setting 
can be tuned by time and/or buffer size. If streaming output is enabled 
aggregated data is pushed to a Redis stream.  You can start the entire service 
with `docker-compose up`.

## Usage

This task is configured with the following environment variables:

```bash
REDIS_HOST              # redis host
REDIS_PORT              # redis port
REDIS_POOL_SIZE         # initial redis connection pool size
REDIS_POOL_SCALE_FACTOR # used to configure connection pool scaling up behavior (REDIS_POOL_SIZE * REDIS_POOL_SCALE_FACTOR)
FLUSH_INTERVAL          # the number of seconds you want the buffer to be flushed
FLUSH_SIZE              # the number of microseconds you want the buffer to fill before flush
RUN_ONCE                # used for unit testing to not start the http server
REDIS_STREAM_OUT        # bool enables redis streaming out
REDIS_STREAM_NAME       # used if streaming out enabled
```

## Example

```
curl -X POST \
  http://127.0.0.1:3000/encounter/testencounter123 \
  -H 'content-type: application/json' \
  -d '{ "data": {"first_name": "Jasmine", "last_name": "Bourbon", "admit_date": "2020-07-20", "middle_name": "Tiffany"}}'
```


## Contributing

* `make run` - runs the api in a docker container
* `make build` - builds your raggs docker container
* `make test` - run unit tests
