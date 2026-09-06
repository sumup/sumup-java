# Events with the JDK HTTP server

```sh
export SUMUP_API_KEY="your_api_key"
export SUMUP_EVENT_SECRET="your_endpoint_signing_secret"
./gradlew :examples:events:run
```

Forward event deliveries to `POST http://localhost:8080/events`.
The example verifies the raw body, runs typed callbacks, and returns HTTP 204 after successful processing.
Invalid deliveries return 400; callback failures return 500 so delivery can be retried.

Use event IDs to deduplicate processing. Resource fetches return the current state;
deleted resources may return an API error. Configure body limits in your production web server or proxy.
