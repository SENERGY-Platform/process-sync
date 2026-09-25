synchronises process states with a remote camunda instance by communicating with a github.com/SENERGY-Platform/mgw-process-sync-client via mqtt 

deploymentIds are overwritten by camunda but they can be correlated by using `GET /metadata/{networkId}?deployment_id=foo` or `GET /metadata/{networkId}?camunda_deployment_id=foo`

## MQTT Config via ENV
you can configure multiple mqtt brokers by using the following ENV variables:
- MQTT_BROKER_{key}
- MQTT_CLIENT_ID_{key}
- MQTT_USER_{key}
- MQTT_PW_{key}

the key is used to group the variables for a specific broker

for backwards compatibility the following ENV variables can be used where the 'key' is inferred as an empty string:
- MQTT_BROKER
- MQTT_CLIENT_ID
- MQTT_USER
- MQTT_PW

## MongoDB Config via ENV

| Env var | Default | Notes |
|---|---|---|
| `MONGO_URL` | `mongodb://localhost:27017` | Full connection string including scheme, passed to the driver unchanged; must not contain credentials. |
| `MONGO_USER` | empty | No authentication when empty. |
| `MONGO_PASSWORD` | empty | Required when `MONGO_USER` is set; never printed at startup and masked when the config is formatted or marshalled. |
| `MONGO_AUTH_SOURCE` | `admin` | Database the user is defined in. |
| `MONGO_DATABASE` | `process_sync` | Must not be empty; collections are configured separately (`MONGO_*_COLLECTION`). |

When `MONGO_USER` is set, the credentials are built from `MONGO_USER`, `MONGO_PASSWORD` and `MONGO_AUTH_SOURCE` alone: they replace any user, password, `authSource` and `authMechanism` given in `MONGO_URL`.

Every applied environment variable is printed at startup, `MONGO_URL` included, so credentials belong in `MONGO_USER`/`MONGO_PASSWORD`, never in `MONGO_URL`. Startup fails unless an authenticated `listCollections` on `MONGO_DATABASE` succeeds within 10 seconds.