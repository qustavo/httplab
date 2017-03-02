# HTTPS Support

HTTPLab does not provides support for HTTPS. In order to decrypt TLS traffic, you can use a proxy like Stunnel.

## How?
```bash
# Generate a self-signed cert
./makecert.sh # Hit Enter until it finishes

# Run Stunnel
./stunnel stunnel.conf
```

Now you can point your HTTP client to :10443.