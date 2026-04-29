FROM debian:trixie-slim

# Add bullseye repo for python2 (removed from trixie)
# Add experimental repo for OpenJDK 25
RUN echo deb http://deb.debian.org/debian/ bullseye main > /etc/apt/sources.list.d/bullseye.list && \
    echo deb http://security.debian.org/debian-security/ bullseye-security main >> /etc/apt/sources.list.d/bullseye.list && \
    echo deb http://deb.debian.org/debian/ experimental main > /etc/apt/sources.list.d/experimental.list

# Base utilities + all apt-installable languages in one layer
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        curl file gcc g++ \
        python3-full python3-pip python3-dev python3-setuptools python3-wheel cython3 \
        libseccomp-dev bzip2 gzip xz-utils unzip \
        python2 tini ca-certificates-java libxtst6 \
        golang nasm ocaml \
        gobjc libgnustep-base-dev \
        mono-mcs mono-runtime \
        jq dirmngr gnupg ca-certificates \
        $([ "$(arch)" = x86_64 ] && echo libc6-dev-i386) && \
    (apt-get install -y --no-install-recommends -t experimental openjdk-25-jdk-headless || \
     apt-get install -y --no-install-recommends openjdk-21-jdk-headless) && \
    mkdir -p /etc/perl && \
    useradd -m judge && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

# OpenJDK 8 (Adoptium Temurin — removed from trixie/sid)
RUN curl -fsSL https://packages.adoptium.net/artifactory/api/gpg/key/public | gpg --dearmor -o /usr/share/keyrings/adoptium.gpg && \
    echo "deb [signed-by=/usr/share/keyrings/adoptium.gpg] https://packages.adoptium.net/artifactory/deb trixie main" > /etc/apt/sources.list.d/adoptium.list && \
    apt-get update && \
    apt-get install -y --no-install-recommends temurin-8-jdk && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

# Rust (via rustup, minimal profile)
RUN runuser judge -c 'curl https://sh.rustup.rs -sSf | sh -s -- -y --profile minimal --default-toolchain stable' && \
    mkdir rust && ( \
        cd rust && \
        curl -sL https://raw.githubusercontent.com/DMOJ/judge-server/master/dmoj/executors/RUST.py | \
            sed '/^CARGO_TOML/,/^"""/!d;//d' > Cargo.toml && \
        mkdir src && \
        curl -sL https://raw.githubusercontent.com/DMOJ/judge-server/master/dmoj/executors/RUST.py | \
            sed '/^TEST_PROGRAM/,/^"""/!d;//d' > src/main.rs && \
        chown -R judge: . && \
        runuser -u judge /home/judge/.cargo/bin/cargo fetch \
    ) && \
    rm -rf rust

# Kotlin
RUN KOTLIN_VER=$(curl -sSL https://api.github.com/repos/JetBrains/kotlin/releases/latest | jq -r .tag_name | sed 's/^v//') && \
    curl -L -okotlin.zip "https://github.com/JetBrains/kotlin/releases/download/v${KOTLIN_VER}/kotlin-compiler-${KOTLIN_VER}.zip" && \
    unzip -q kotlin.zip && mv kotlinc /opt/kotlin && rm kotlin.zip

# Dart SDK
RUN cd /opt && \
    case "$(arch)" in x86_64) DART_ARCH=x64;; aarch64) DART_ARCH=arm64;; *) false;; esac && \
    curl -L -odart.zip "https://storage.googleapis.com/dart-archive/channels/stable/release/$( \
        curl -sSL https://storage.googleapis.com/dart-archive/channels/stable/release/latest/VERSION | jq -r .version \
    )/sdk/dartsdk-linux-$DART_ARCH-release.zip" && \
    unzip dart.zip && find /opt/dart-sdk -type d -exec chmod go+rx {} + && rm dart.zip

ENV PATH="/home/judge/.cargo/bin:/opt/kotlin/bin:/opt/dart-sdk/bin:${PATH}"

# Install DMOJ judge
ARG TAG=master
RUN mkdir /judge /problems && cd /judge && \
    curl -L https://github.com/DMOJ/judge-server/archive/"${TAG}".tar.gz | tar -xz --strip-components=1 && \
    python3 -m venv --prompt=DMOJ /env && \
    /env/bin/pip3 install cython setuptools && \
    /env/bin/pip3 install -e . && \
    /env/bin/python3 setup.py develop && \
    HOME=~judge . ~judge/.profile && \
    runuser -u judge -w PATH -- /env/bin/dmoj-autoconf -V > /judge-runtime-paths.yml && \
    echo '  crt_x86_in_lib32: true' >> /judge-runtime-paths.yml

ENTRYPOINT ["/usr/bin/tini", "--", "/judge/.docker/entry"]
