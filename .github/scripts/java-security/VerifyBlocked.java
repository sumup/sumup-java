import java.io.InputStream;
import java.nio.file.Files;
import java.nio.file.Paths;
import java.security.cert.*;
import java.time.Instant;
import java.util.*;

/** Offline PKIX verification using public certificates only. */
public final class VerifyBlocked {
    public static void main(String[] arguments) throws Exception {
        CertificateFactory factory = CertificateFactory.getInstance("X.509");
        Set<TrustAnchor> anchors = new HashSet<>();
        try (InputStream input = Files.newInputStream(Paths.get(arguments[0]))) {
            for (Certificate certificate : factory.generateCertificates(input)) {
                anchors.add(new TrustAnchor((X509Certificate) certificate, null));
            }
        }
        PKIXParameters parameters = new PKIXParameters(anchors);
        parameters.setRevocationEnabled(false);
        parameters.setDate(Date.from(Instant.parse("2026-10-22T00:00:00Z")));
        for (TrustAnchor anchor : anchors) {
            CertPath control = factory.generateCertPath(Collections.singletonList(anchor.getTrustedCert()));
            CertPathValidator.getInstance("PKIX").validate(control, parameters);
        }
        boolean expectBlocked = arguments[2].equals("blocked");
        int checked = 0;
        try (InputStream input = Files.newInputStream(Paths.get(arguments[1]))) {
            for (Certificate certificate : factory.generateCertificates(input)) {
                CertPath path = factory.generateCertPath(Collections.singletonList(certificate));
                try {
                    CertPathValidator.getInstance("PKIX").validate(path, parameters);
                    if (expectBlocked) throw new IllegalStateException("Compromised certificate was accepted");
                } catch (CertPathValidatorException exception) {
                    if (!expectBlocked || !exception.getMessage().contains("Untrusted certificate")) {
                        throw exception;
                    }
                }
                checked++;
            }
        }
        if (checked != 16) throw new IllegalStateException("Expected 16 certificates, got " + checked);
        System.out.println("PASS: " + checked + " certificates " + (expectBlocked ? "blocked" : "accepted (baseline)"));
    }
}
