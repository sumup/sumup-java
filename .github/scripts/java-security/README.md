Public certificate fixtures from https://emudhra.we-have-a-problem.com/
(impacted-intermediates.json and roots.json), referenced by https://ops.java/emudhra.

The verifier uses offline PKIX validation at a fixed date. It requires all 16
compromised certificates to fail specifically as untrusted and checks the four
unblocked issuer roots as controls. No private keys or network requests are used.
