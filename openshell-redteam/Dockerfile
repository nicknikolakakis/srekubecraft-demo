# minimal demo target for the redteam harness, no agent code, just the shape of a caged python process
FROM debian:trixie-slim

# iproute2 is required by the OpenShell supervisor for network namespaces
RUN apt-get update && apt-get install -y --no-install-recommends \
      python3 iproute2 curl ca-certificates bash coreutils \
 && rm -rf /var/lib/apt/lists/* \
 && { [ -x /usr/bin/python3.13 ] || ln -s "$(readlink -f "$(command -v python3)")" /usr/bin/python3.13; }

# a venv-style symlink chain to the allowed binary, so technique 6 has something to resolve
RUN install -d /opt/app/.venv/bin \
 && ln -s /usr/bin/python3.13 /opt/app/.venv/bin/python3.13

# non-root, matches the policy run_as_user
RUN groupadd -g 10000 app && useradd -u 10000 -g 10000 -M -d /opt/data app \
 && install -d -o app -g app /opt/data

USER app
WORKDIR /opt/data
