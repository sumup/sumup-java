<div align="center">

# SumUp Java SDK

[![Maven Central](https://img.shields.io/maven-central/v/com.sumup/sumup-sdk)](https://central.sonatype.com/artifact/com.sumup/sumup-sdk)
[![Documentation][docs-badge]](https://developer.sumup.com)
[![CI Status](https://github.com/sumup/sumup-java/actions/workflows/ci.yaml/badge.svg)](https://github.com/sumup/sumup-java/actions/workflows/ci.yaml)
[![License](https://img.shields.io/github/license/sumup/sumup-java)](./LICENSE)

</div>

_**IMPORTANT:** This SDK is under heavy development and subject to breaking changes._

The Java SDK for the SumUp [API](https://developer.sumup.com) generated from the canonical OpenAPI specification. Requires Java 17 or newer.

## Installation

### Gradle (Kotlin DSL)

Add the dependency in your `build.gradle.kts` file:

```kotlin
dependencies {
    implementation("com.sumup:sumup-sdk:0.0.7")
}
```

### Gradle (Groovy)

Add the dependency in your `build.gradle` file:

```groovy
dependencies {
    implementation 'com.sumup:sumup-sdk:0.0.7'
}
```

### Maven

Add the dependency in your `pom.xml` file:

```xml
<dependency>
  <groupId>com.sumup</groupId>
  <artifactId>sumup-sdk</artifactId>
  <version>0.0.7</version>
</dependency>
```

## Getting Started

Authenticate with a personal access token before making calls:

```bash
export SUMUP_API_KEY="my-token"
```

Create clients using the default constructor (reads `SUMUP_API_KEY`):

```java
SumUpClient client = new SumUpClient();
SumUpAsyncClient asyncClient = new SumUpAsyncClient();
```

Or pass the API key explicitly:

```java
SumUpClient client = new SumUpClient("api-key");
```

If you need more configuration, use the builder (for example, to override the base URL):

```java
SumUpClient client =
    SumUpClient.builder()
        .baseUri("https://api.sumup.com")
        .build();
```

## Usage

### Creating an Online Checkout

Synchronous example:

```java
import com.sumup.sdk.SumUpClient;
import com.sumup.sdk.models.CheckoutCreateRequest;
import com.sumup.sdk.models.Currency;
import java.util.UUID;

SumUpClient client =
    SumUpClient.builder()
        .build();

String merchantCode = client.merchant().getMerchantProfile().merchantCode();
String checkoutReference = "checkout-" + UUID.randomUUID();

CheckoutCreateRequest request =
    CheckoutCreateRequest.builder()
        .amount(10.00f)
        .currency(Currency.EUR)
        .checkoutReference(checkoutReference)
        .merchantCode(merchantCode)
        .description("Test payment")
        .redirectUrl("https://example.com/success")
        .returnUrl("https://example.com/webhook")
        .build();

var checkout = client.checkouts().createCheckout(request);
System.out.printf("Checkout %s created with id %s%n", checkout.checkoutReference(), checkout.id());
```

Asynchronous example:

```java
import com.sumup.sdk.SumUpAsyncClient;
import com.sumup.sdk.models.CheckoutCreateRequest;
import com.sumup.sdk.models.Currency;
import java.util.UUID;
import java.util.concurrent.CompletableFuture;

SumUpAsyncClient client =
    SumUpAsyncClient.builder()
        .build();

CompletableFuture<Void> checkoutFuture =
    client
        .merchant()
        .getMerchantProfile()
        .thenApply(profile -> profile.merchantCode())
        .thenCompose(
            merchantCode -> {
              String checkoutReference = "checkout-" + UUID.randomUUID();
              CheckoutCreateRequest request =
                  CheckoutCreateRequest.builder()
                      .amount(10.00f)
                      .currency(Currency.EUR)
                      .checkoutReference(checkoutReference)
                      .merchantCode(merchantCode)
                      .description("Test payment")
                      .redirectUrl("https://example.com/success")
                      .returnUrl("https://example.com/webhook")
                      .build();

              return client.checkouts().createCheckout(request);
            })
        .thenAccept(
            checkout ->
                System.out.printf(
                    "Checkout %s created (%s)%n",
                    checkout.checkoutReference(), checkout.id()));

checkoutFuture.join();
```

### Creating a Card Reader Checkout

Synchronous example:

```java
import com.sumup.sdk.SumUpClient;
import com.sumup.sdk.models.CreateReaderCheckoutRequest;
import com.sumup.sdk.models.Money;
import java.util.Optional;
import java.util.UUID;

String merchantCode = System.getenv("SUMUP_MERCHANT_CODE");

SumUpClient client =
    SumUpClient.builder()
        .build();

String readerId =
    Optional.ofNullable(System.getenv("SUMUP_READER_ID"))
        .orElseGet(
            () ->
                client.readers().listReaders(merchantCode).items().stream()
                    .findFirst()
                    .map(reader -> reader.id().value())
                    .orElseThrow(() -> new IllegalStateException("No paired readers found.")));

CreateReaderCheckoutRequest request =
    CreateReaderCheckoutRequest.builder()
        .description("sumup-java reader checkout " + UUID.randomUUID())
        .totalAmount(
            Money.builder()
                .currency("EUR")
                .minorUnit(2L)
                .value(1000L) // €10.00
                .build())
        .returnUrl("https://example.com/webhook")
        .build();

client.readers().createReaderCheckout(merchantCode, readerId, request);
System.out.println("Reader checkout created.");
```

Asynchronous example:

```java
import com.sumup.sdk.SumUpAsyncClient;
import com.sumup.sdk.models.CreateReaderCheckoutRequest;
import com.sumup.sdk.models.Money;
import java.util.Optional;
import java.util.UUID;
import java.util.concurrent.CompletableFuture;

String merchantCode = System.getenv("SUMUP_MERCHANT_CODE");

SumUpAsyncClient client =
    SumUpAsyncClient.builder()
        .build();

CompletableFuture<String> readerIdFuture =
    Optional.ofNullable(System.getenv("SUMUP_READER_ID"))
        .map(CompletableFuture::completedFuture)
        .orElseGet(
            () ->
                client
                    .readers()
                    .listReaders(merchantCode)
                    .thenApply(
                        response ->
                            response.items().stream()
                                .findFirst()
                                .map(reader -> reader.id().value())
                                .orElseThrow(
                                    () ->
                                        new IllegalStateException("No paired readers found."))));

readerIdFuture
    .thenCompose(
        readerId -> {
          CreateReaderCheckoutRequest request =
              CreateReaderCheckoutRequest.builder()
                  .description("sumup-java reader checkout " + UUID.randomUUID())
                  .totalAmount(
                      Money.builder()
                          .currency("EUR")
                          .minorUnit(2L)
                          .value(1000L)
                          .build())
                  .returnUrl("https://example.com/webhook")
                  .build();

          return client.readers().createReaderCheckout(merchantCode, readerId, request);
        })
    .thenAccept(
        response ->
            System.out.printf(
                "Reader checkout created: %s%n",
                Optional.ofNullable(response.data())
                    .map(data -> data.clientTransactionId())
                    .orElse("<unknown>")))
    .join();
```

## Examples

- `examples/basic` – lists recent checkouts to verify that your API token works.
- `examples/card-reader-checkout` – lists paired readers and creates a €10 checkout on the first available device.

To run the card reader example locally:

```bash
export SUMUP_API_KEY="your_api_key"
export SUMUP_MERCHANT_CODE="your_merchant_code"
# Optional: pick a concrete reader instead of taking the first
# export SUMUP_READER_ID="your_reader_id"
./gradlew :examples:card-reader-checkout:run
```

## Generating Javadoc

Build the aggregated API reference locally with `just javadoc` (or `./gradlew aggregateJavadoc`). The HTML output is placed under `build/docs/javadoc/index.html`.

[docs-badge]: https://img.shields.io/badge/SumUp-documentation-white.svg?logo=data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjQiIGhlaWdodD0iMjQiIHZpZXdCb3g9IjAgMCAyNCAyNCIgZmlsbD0ibm9uZSIgY29sb3I9IndoaXRlIiB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciPgogICAgPHBhdGggZD0iTTIyLjI5IDBIMS43Qy43NyAwIDAgLjc3IDAgMS43MVYyMi4zYzAgLjkzLjc3IDEuNyAxLjcxIDEuN0gyMi4zYy45NCAwIDEuNzEtLjc3IDEuNzEtMS43MVYxLjdDMjQgLjc3IDIzLjIzIDAgMjIuMjkgMFptLTcuMjIgMTguMDdhNS42MiA1LjYyIDAgMCAxLTcuNjguMjQuMzYuMzYgMCAwIDEtLjAxLS40OWw3LjQ0LTcuNDRhLjM1LjM1IDAgMCAxIC40OSAwIDUuNiA1LjYgMCAwIDEtLjI0IDcuNjlabTEuNTUtMTEuOS03LjQ0IDcuNDVhLjM1LjM1IDAgMCAxLS41IDAgNS42MSA1LjYxIDAgMCAxIDcuOS03Ljk2bC4wMy4wM2MuMTMuMTMuMTQuMzUuMDEuNDlaIiBmaWxsPSJjdXJyZW50Q29sb3IiLz4KPC9zdmc+
