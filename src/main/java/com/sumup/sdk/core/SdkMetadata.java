package com.sumup.sdk.core;

import java.io.IOException;
import java.io.InputStream;
import java.nio.charset.StandardCharsets;
import java.util.Map;

/**
 * Provides metadata about the SDK that can be attached to outgoing requests, such as the current
 * version and the default {@code User-Agent} header value.
 */
public final class SdkMetadata {
  private static final String API_VERSION_RESOURCE = "/com/sumup/sdk/api-version.txt";
  private static final String VERSION_RESOURCE = "/com/sumup/sdk/sdk-version.txt";
  private static final String USER_AGENT_PREFIX = "sumup-java";
  private static final String LANGUAGE = "java";

  private static final String API_VERSION = loadResource(API_VERSION_RESOURCE);
  private static final String VERSION = loadResource(VERSION_RESOURCE);
  private static final String USER_AGENT = USER_AGENT_PREFIX + "/v" + VERSION;
  private static final Map<String, String> RUNTIME_HEADERS =
      Map.of(
          "X-Sumup-Api-Version", API_VERSION,
          "X-Sumup-Lang", LANGUAGE,
          "X-Sumup-Package-Version", VERSION,
          "X-Sumup-OS", System.getProperty("os.name", "unknown"),
          "X-Sumup-Arch", runtimeArch(),
          "X-Sumup-Runtime", runtimeIdentifier(),
          "X-Sumup-Runtime-Version", Runtime.version().toString());

  private SdkMetadata() {}

  /** Returns the API version declared by the OpenAPI specification. */
  public static String apiVersion() {
    return API_VERSION;
  }

  /** Returns the SDK version read from the generated version resource. */
  public static String version() {
    return VERSION;
  }

  /** Returns the {@code User-Agent} header value that should be sent with each request. */
  public static String userAgent() {
    return USER_AGENT;
  }

  /** Returns the runtime headers that should be sent with each request. */
  public static Map<String, String> runtimeHeaders() {
    return RUNTIME_HEADERS;
  }

  static String runtimeArch() {
    String arch = System.getProperty("os.arch", "unknown").toLowerCase();
    return switch (arch) {
      case "amd64", "x86_64" -> "x86_64";
      case "x86", "i386", "i486", "i586", "i686" -> "x86";
      case "aarch64", "arm64" -> "arm64";
      case "arm", "armv7", "armv7l" -> "arm";
      default -> arch;
    };
  }

  private static String runtimeIdentifier() {
    return LANGUAGE + Runtime.version();
  }

  private static String loadResource(String path) {
    try (InputStream stream = SdkMetadata.class.getResourceAsStream(path)) {
      if (stream == null) {
        return "unknown";
      }
      return new String(stream.readAllBytes(), StandardCharsets.UTF_8).trim();
    } catch (IOException e) {
      return "unknown";
    }
  }
}
