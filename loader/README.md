# Loader

Loader is a very simple tool used to create load on a pilosa instance.  It has a simple `toml` config to create tasks. It currently only reports success/error for calls.  It purposely doesn't track stats.  Stats should be monitored from the pilosa stats endpoints, such as the `/metrics` endpoint that feed prometheus, etc.

## Config

A sample config would look like this:

```toml
[[Tasks]]
  Connections = 10
  Delay = "10ms"
  Query = "TopN(model, n=5)"
  URL = "http://localhost:10101/index/equipment/query"

[[Tasks]]
  Connections = 1
  Delay = "1s"
  Query = "TopN(model, n=5)"
  URL = "http://localhost:10101/index/equipment/query"
```

You can have as many tasks as you want, but you need at least one for the program to create any load.

### Definitions

| Variable | Definition |
| --- | --- |
| Connections | Number of connections for this task to use. If you specify `10`, it means that it will create `10` actual connections for this task |
| Delay | This is the amount of time between each request per connection.  If `1s` is specified, then it will delay for `1s` between each request |
| Query | The desired query to execute |
| URL | The query endoint to post the query to |

The program will try to stagger the requests per task if more than one connection is specified.  For example, if you specify `10` connections, and a delay of `10ms`, then it will fire one request every `1ms`, for a total of `10` requests every `10ms`.

## Usage

To get a template for a new config, you can issue the following command:

```sh
loader config
```

To run the program with a specific config:

```sh
loader -config ./loader/sample.config.toml
```
