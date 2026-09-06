package com.sumup.sdk.events;

import java.nio.charset.StandardCharsets;
import java.security.GeneralSecurityException;
import java.security.MessageDigest;
import java.time.Instant;
import java.util.HexFormat;
import javax.crypto.Mac;
import javax.crypto.spec.SecretKeySpec;

/** Verifies event signatures without decoding or modifying the request body. */
public final class EventSignature {
  /** Header containing the delivery timestamp and signature. */
  public static final String HEADER_NAME = "X-SumUp-Webhook-Signature";

  private EventSignature() {}

  /**
   * Verifies the signature and enforces a five-minute delivery window in either direction.
   *
   * @param body original HTTP request bytes; do not reserialize the JSON
   * @param signature complete signature header value
   * @param secret endpoint signing secret, not an API key
   * @throws EventSignatureException if the header is missing, malformed, or does not match
   * @throws EventSignatureExpiredException if the delivery timestamp is outside the window
   * @throws IllegalArgumentException if the secret is blank
   */
  public static void verify(byte[] body, String signature, String secret) {
    verify(body, signature, secret, Instant.now().getEpochSecond());
  }

  static void requireSecret(String secret) {
    if (secret == null || secret.isBlank())
      throw new IllegalArgumentException("An endpoint signing secret is required.");
  }

  static void verify(byte[] body, String signature, String secret, long now) {
    requireSecret(secret);
    var parts = signature == null ? new String[0] : signature.trim().split(",", -1);
    if (parts.length != 2 || !parts[0].startsWith("t=") || !parts[1].startsWith("v1="))
      throw new EventSignatureException("Expected t=<timestamp>,v1=<signature>.");
    var timestampText = parts[0].substring(2);
    long timestamp;
    byte[] digest;
    try {
      if (timestampText.isEmpty()
          || !timestampText.chars().allMatch(c -> c >= '0' && c <= '9')
          || parts[1].length() != 67) throw new IllegalArgumentException();
      timestamp = Long.parseLong(timestampText);
      digest = HexFormat.of().parseHex(parts[1].substring(3));
    } catch (IllegalArgumentException cause) {
      throw new EventSignatureException("Invalid signature timestamp or digest.");
    }
    try {
      var mac = Mac.getInstance("HmacSHA256");
      mac.init(new SecretKeySpec(secret.getBytes(StandardCharsets.UTF_8), "HmacSHA256"));
      mac.update(("v1:" + timestampText + ":").getBytes(StandardCharsets.UTF_8));
      if (!MessageDigest.isEqual(mac.doFinal(body), digest))
        throw new EventSignatureException("The event signature does not match.");
    } catch (GeneralSecurityException cause) {
      throw new IllegalStateException("Cannot compute an event signature.", cause);
    }
    if (timestamp < now - 300 || timestamp > now + 300)
      throw new EventSignatureExpiredException(
          "The signature timestamp is outside the five-minute window.");
  }
}
