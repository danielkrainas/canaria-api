# Change Log
All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).

## [Unreleased]
### Added
- Basic webhook support (`/v1/canary/<canary_id>/webhook`).
- Health and version check endpoint (`/v1`).
- Webhook events for `ping` and `dead`.
- Webhook test endpoint (`/v1/canary/<canary_id>/webhook/<webhook_id>/ping`)
- YAML config file support.
- Silly authentication support.
- htpasswd authentication with bcrypt format password support.
- TLS support.
- TLS with LetsEncrypt support.

## [0.0.1-alpha] - 2016-04-14
### Added
- Basic canary support.
- Memory storage support.
