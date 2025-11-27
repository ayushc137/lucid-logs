# =============================================================================
# Dev Container Dockerfile
# =============================================================================
# This creates a development environment with all tools pre-installed.
# Based on the official Rust devcontainer image.
# =============================================================================

FROM mcr.microsoft.com/devcontainers/rust:1-1-bookworm

# Install additional system packages
RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
    && apt-get -y install --no-install-recommends \
        # Build essentials
        build-essential \
        pkg-config \
        libssl-dev \
        # Database tools
        postgresql-client \
        # Utilities
        curl \
        wget \
        jq \
        httpie \
        # Git tools
        git-lfs \
    && apt-get autoremove -y \
    && apt-get clean -y \
    && rm -rf /var/lib/apt/lists/*

# Install Rust tools as the vscode user
USER vscode

# Install commonly used cargo tools
RUN cargo install --locked \
    cargo-watch \
    cargo-nextest \
    cargo-audit \
    cargo-deny \
    cargo-machete \
    cargo-expand \
    bacon \
    lefthook \
    cargo-outdated \
    cargo-edit

# Install sccache for faster builds (optional but recommended)
RUN cargo install sccache

# Install SurrealDB CLI
RUN curl -sSf https://install.surrealdb.com | sh -s -- --nightly

# Set up cargo environment
ENV CARGO_HOME=/workspace/.cargo
ENV RUSTUP_HOME=/workspace/.rustup
ENV PATH="${CARGO_HOME}/bin:${RUSTUP_HOME}/bin:${PATH}"

# Enable sccache
ENV RUSTC_WRAPPER=sccache

# Set working directory
WORKDIR /workspace

USER root

