package com.sumup.examples.cardreader;

import com.sumup.sdk.SumUpClient;
import com.sumup.sdk.core.ApiException;
import com.sumup.sdk.models.CreateReaderCheckoutRequest;
import com.sumup.sdk.models.Money;
import java.util.Optional;
import java.util.UUID;

/** Demonstrates how to create a card-present checkout on a reader using the generated SDK. */
public final class CardReaderCheckoutExample {

  private CardReaderCheckoutExample() {}

  public static void main(String[] args) {
    String merchantCode = requireEnv("SUMUP_MERCHANT_CODE");

    SumUpClient client = new SumUpClient();

    Optional<String> readerId =
        client.readers().list(merchantCode).items().stream()
            .findFirst()
            .map(reader -> reader.id().value());
    if (readerId.isEmpty()) {
      System.err.println("Merchant has no paired readers.");
      return;
    }

    String checkoutReference = "checkout-" + UUID.randomUUID();
    System.out.printf(
        "Creating checkout %s on reader %s%n", checkoutReference, readerId.orElse("<unknown>"));

    if (createReaderCheckout(client, merchantCode, readerId.get(), checkoutReference)) {
      System.out.println("✓ Checkout created successfully!");
    } else {
      System.err.println("✗ Failed to create checkout");
    }
  }

  private static boolean createReaderCheckout(
      SumUpClient client, String merchantCode, String readerId, String checkoutReference) {
    CreateReaderCheckoutRequest request =
        CreateReaderCheckoutRequest.builder()
            .description("sumup-java card reader checkout " + checkoutReference)
            .totalAmount(Money.builder().currency("EUR").minorUnit(2L).value(1000L).build())
            .build();

    try {
      client.readers().createCheckout(merchantCode, readerId, request);
      return true;
    } catch (ApiException ex) {
      System.err.printf(
          "Checkout creation failed (%d): %s%n", ex.getStatusCode(), ex.getResponseBody());
      return false;
    }
  }

  private static String requireEnv(String name) {
    String value = System.getenv(name);
    if (value == null || value.isBlank()) {
      throw new IllegalStateException(name + " environment variable must be set");
    }
    return value;
  }
}
