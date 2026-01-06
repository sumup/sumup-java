package com.sumup.examples.basic;

import com.sumup.sdk.SumUpClient;
import com.sumup.sdk.SumUpEnvironment;
import com.sumup.sdk.core.ApiException;
import java.util.List;

/**
 * Demonstrates a minimal SumUp client that lists recent checkouts.
 *
 * <p>Set {@code SUMUP_API_KEY} to a valid personal access token, then run {@code ./gradlew
 * :examples:basic:run}.
 */
public final class BasicExample {

  private BasicExample() {}

  public static void main(String[] args) {
    String apiKey = System.getenv("SUMUP_API_KEY");
    if (apiKey == null || apiKey.isBlank()) {
      System.err.println("SUMUP_API_KEY environment variable must be set.");
      return;
    }

    SumUpClient client =
        SumUpClient.builder().environment(SumUpEnvironment.PRODUCTION).accessToken(apiKey).build();

    try {
      List<com.sumup.sdk.models.CheckoutSuccess> checkouts = client.checkouts().listCheckouts();
      System.out.printf("Fetched %d checkouts.%n", checkouts.size());
    } catch (ApiException ex) {
      System.err.printf(
          "API call failed with status %d: %s%n", ex.getStatusCode(), ex.getResponseBody());
    }
  }
}
