package com.sumup.sdk.core;

import java.io.IOException;
import java.io.InputStream;
import java.nio.charset.StandardCharsets;

/**
 * Provides metadata about the SDK that can be attached to outgoing requests, such as the current
 * version and the default {@code User-Agent} header value.
 */
public final class SdkMetadata {
  private static final String VERSION_RESOURCE = "/com/sumup/sdk/sdk-version.txt";
  private static final String USER_AGENT_PREFIX = "sumup-java";

  private static final String VERSION = loadVersion();
  private static final String USER_AGENT = USER_AGENT_PREFIX + "/v" + VERSION;

  private SdkMetadata() {}

  /** Returns the SDK version read from the generated version resource. */
  public static String version() {
    return VERSION;
  }

  /** Returns the {@code User-Agent} header value that should be sent with each request. */
  public static String userAgent() {
    return USER_AGENT;
  }

  private static String loadVersion() {
    try (InputStream stream = SdkMetadata.class.getResourceAsStream(VERSION_RESOURCE)) {
      if (stream == null) {
        return "unknown";
      }
      return new String(stream.readAllBytes(), StandardCharsets.UTF_8).trim();
    } catch (IOException e) {
      return "unknown";
    }
  }
}
