package com.sumup.examples.basic;

import com.sumup.sdk.SumUpClient;
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
    SumUpClient client = new SumUpClient();

    try {
      List<com.sumup.sdk.models.CheckoutSuccess> checkouts = client.checkouts().list();
      System.out.printf("Fetched %d checkouts.%n", checkouts.size());
    } catch (ApiException ex) {
      System.err.printf(
          "API call failed with status %d: %s%n", ex.getStatusCode(), ex.getResponseBody());
    }
  }
}
