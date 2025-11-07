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