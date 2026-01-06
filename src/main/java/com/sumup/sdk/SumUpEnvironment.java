package com.sumup.sdk;

/** Represents the available SumUp API environments. */
public enum SumUpEnvironment {
  PRODUCTION("https://api.sumup.com"),
  SANDBOX("https://api.sumup.com");

  private final String baseUrl;

  SumUpEnvironment(String baseUrl) {
    this.baseUrl = baseUrl;
  }

  public String getBaseUrl() {
    return baseUrl;
  }
}
