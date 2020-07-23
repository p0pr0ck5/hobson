hobson
======

Failover DNS for Consul SD

# Table of Contents

* [Overview](#overview)
* [Usage](#usage)
* [License](#license)

# Overview

hobson is a simply utility to provide a failover DNS strategy for Consul DNS
resolution service discovery. Currently, Consul's DNS authoritative DNS server
provides no mechanism to return one or a subset of DNS records for a given
service. hobson will watch predefined registered Consul services and respond
to DNS queries with a single A record for a given name, based on the list of
service registrations. Only addresses of services with passing health checks are
considered in the list of records to return, and for any given list of
addresses, the same address will be returned every time.

# Usage

hobson configuration is provided via YAML config file, the path of which is
referenced by the `-config` CLI flag:

```bash
$ ./hobson -config ./hobson.yaml
```

The config file expects the following elements:

* **bind**: The address and port to which to bind the DNS server.
* **prometheus_bind**: The address and port to which to bind the Prometheus metrics exposition HTTP endpoint.
* **zone**: The zone under which to service DNS names.
* **services**: A list of Consul service names to watch and return records for.

Note that hobson currently relies on the Consul Go SDK for discovering where
to contact a Consul agent; see the [Consul documentation](https://www.consul.io/docs/commands/index.html#environment-variables)
for details on this behavior.

# License

Copyright 2019 Robert Paprocki.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

