# DNS Redirect Service

This Go service is designed to dynamically handle HTTP requests based on CNAME DNS records. It performs DNS lookups to retrieve CNAME records and redirects the request based on specific suffixes defined via environment variables. The redirect status code (e.g., 301, 302, etc.) is also determined dynamically, making the service flexible for a wide variety of use cases.

## Features
- Redirects requests based on CNAME records.
- Supports multiple types of HTTP status codes (301, 302, 303, 307, 308).
- DNS configuration can be customized using `/etc/resolv.conf` or overridden via an environment variable.
- Uses Google Public DNS as a fallback if system DNS configuration fails.

## Requirements
- Go 1.16+
- Network connectivity to perform DNS lookups.

## Environment Variables
The service uses environment variables to determine which domains to redirect and with which HTTP status codes. The following environment variables are supported:

- `DNS_CONFIG_FILE`: Path to the DNS configuration file (default: `/etc/resolv.conf`).
- `REDIRECT_DOMAIN_301`: Domain suffix to be redirected with HTTP status code 301 (Moved Permanently).
- `REDIRECT_DOMAIN_302`: Domain suffix to be redirected with HTTP status code 302 (Found).
- `REDIRECT_DOMAIN_303`: Domain suffix to be redirected with HTTP status code 303 (See Other).
- `REDIRECT_DOMAIN_307`: Domain suffix to be redirected with HTTP status code 307 (Temporary Redirect).
- `REDIRECT_DOMAIN_308`: Domain suffix to be redirected with HTTP status code 308 (Permanent Redirect).

> **Note**: At least one of the `REDIRECT_DOMAIN_` environment variables must be defined, or the service will exit with an error.

## How It Works
1. The service listens on port `8080` and handles incoming HTTP requests.
2. For each request, it performs a CNAME lookup for the hostname.
3. Based on the CNAME result, it determines if the CNAME ends with one of the configured suffixes (e.g., `example.com`).
4. If a match is found, the service removes the suffix and redirects the request to the target domain with the appropriate status code.
5. The scheme (`http` or `https`) is preserved from the original request.

## Running the Service
To run the service:

1. Clone the repository and navigate to the project directory.
2. Set the required environment variables:
   ```sh
   export REDIRECT_DOMAIN_301="permanent-redirect.com"
   export REDIRECT_DOMAIN_307="temporary-redirect.com"
   ```
3. Build and run the service:
   ```sh
   go build -o dynamic-cname-redirect
   ./dynamic-cname-redirect
   ```

## Example Usage
- Suppose you have a CNAME record like `foo.example.com` that points to `bar.example.net.permanent-redirect.com`.
- If `REDIRECT_DOMAIN_301="permanent-redirect.com"` is set, and the CNAME matches, the service will remove `.permanent-redirect.com` and redirect `foo.example.com` to `bar.example.net` using a `301 Moved Permanently` status code.

## Logging
- The service logs all redirects, including the source URL, the target URL, and the HTTP status code used.
- Errors related to DNS lookups or unexpected CNAME formats are also logged.

## Error Handling
- If no CNAME is found or if the format of the CNAME is not as expected, the service returns an `HTTP 500 Internal Server Error` response.
- If no `REDIRECT_DOMAIN_` environment variables are defined, the service exits with an error.

## License
This project is licensed under the MIT License. See the LICENSE file for more details.

## Contributions
Contributions are welcome! Feel free to submit a pull request or open an issue for feature requests and bug reports.

## Author
Arnold Daniels - [Jasny](https://jasny.net)
