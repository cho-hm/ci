# CI

Go 기반 GitHub Actions CI 도구. Gradle/Node.js 프로젝트의 Docker 빌드(GHCR) + 패키지 배포(GPR)를 단일 바이너리로 처리한다.

## 사용법

GitHub Releases에서 크로스컴파일된 바이너리를 다운로드하여 workflow에서 실행한다.

```bash
./ci -env gradle -parse -check -publish -build
```

### 플래그

| 플래그 | 설명 |
|--------|------|
| `-env` | 빌드 환경 (`gradle`, `node`). 기본값: `gradle` |
| `-parse` | parse만 실행 |
| `-check` | parse + check |
| `-publish` | parse + check + publish (GPR 패키지 배포) |
| `-build` | parse + check + build (Docker 빌드 + GHCR push) |

`-publish`와 `-build`는 동시 사용 가능. 각각 독립 실행되며, 하나라도 실패하면 exit 1.

### 프로퍼티 파일

프로젝트 루트에 배치. 아래는 전체 키와 기본값이며, 필요한 것만 오버라이드하면 된다.

| 파일 | 용도 | 필요 시점 |
|------|------|-----------|
| `ci-ghcr.properties` | Docker 빌드 설정 | `-build` 사용 시 |
| `ci-mvn.properties` | Maven publish 설정 | `-publish` + `env=gradle` |
| `ci-npm.properties` | npm publish 설정 | `-publish` + `env=node` |

**ci-ghcr.properties** (Docker/GHCR 빌드)
```properties
trigger.type=signed-tag
trigger.branch=master:develop
docker.file.path=./.github/Dockerfile
build.command=./gradlew clean test bootJar --no-daemon  # env=node이면 "npm ci && npm run build"
image.platform=linux/amd64,linux/arm64
image.name.suffix=trigger-type:tag:branch:sha:short-sha:latest
gpg.repo.url=https://github.com/org/gpg-keys.git
gpg.repo.gpg.path=keys/gpg
gpg.repo.asc.path=keys/asc
gpg.repo.branch=master
```

**ci-mvn.properties** (Gradle Maven publish)
```properties
trigger.type=signed-tag
trigger.branch=master
publish.command=./gradlew clean test publish --no-daemon
```

**ci-npm.properties** (Node.js npm publish)
```properties
trigger.type=signed-tag
trigger.branch=master
publish.command=npm publish --registry=https://npm.pkg.github.com
```

구분자: `=`, 주석: `#` (따옴표 내부 `#`은 주석 아님), `trigger.branch`는 `:`로 여러 브랜치 지정 가능.

`trigger.type` 유효값: `signed-tag` (GPG 서명 태그), `tag` (일반 태그), `branch` (브랜치 push)

### Secrets

Workflow의 `env`로 전달해야 하는 변수:

| 변수 | 필수 | 설명 |
|------|------|------|
| `GITHUB_TOKEN` | O | workflow에서 `${{ secrets.GITHUB_TOKEN }}`으로 전달 |
| `PAT` | - | Personal Access Token. 없으면 `GITHUB_TOKEN` 사용 |
| `GPG_TOKEN` | - | `signed-tag` 트리거 시 GPG repo 접근용 |

`GITHUB_WORKSPACE`, `GITHUB_REF_TYPE`, `GITHUB_REF_NAME`, `GITHUB_SHA`, `GITHUB_REPOSITORY`, `GITHUB_ACTOR` 등은 GitHub Actions가 자동 제공하므로 별도 설정 불필요.

### 토큰 발급 및 권한 설정

#### GITHUB_TOKEN

workflow에서 자동 제공되는 토큰. `permissions`로 필요 권한을 선언한다.

```yaml
permissions:
  contents: read      # 코드 checkout
  packages: write     # GHCR push, GPR publish
```

별도 발급 불필요. workflow 파일에 위 permissions를 명시하면 된다.

#### PAT (Personal Access Token)

GHCR 로그인에 사용. 없으면 `GITHUB_TOKEN`으로 대체된다. 별도 발급이 필요한 경우:

1. GitHub → Settings → Developer settings → **Personal access tokens** → **Fine-grained tokens** → Generate new token
2. 필요 권한:
   - **Repository access**: 대상 리포지토리 선택
   - **Permissions**:
     - `Read access to metadata`
     - `Read and Write access to packages` (GHCR push)
3. 생성된 토큰을 대상 리포지토리의 Settings → Secrets and variables → Actions → **New repository secret**에 `PAT`로 등록

#### GPG_TOKEN

`signed-tag` 트리거 시 GPG 키가 저장된 private repo를 clone하기 위한 토큰.

1. GitHub → Settings → Developer settings → **Personal access tokens** → **Fine-grained tokens** → Generate new token
2. 필요 권한:
   - **Repository access**: GPG 키 저장 리포지토리 선택
   - **Permissions**:
     - `Read access to metadata`
     - `Read access to contents` (repo clone)
3. 생성된 토큰을 대상 리포지토리의 Settings → Secrets and variables → Actions → **New repository secret**에 `GPG_TOKEN`으로 등록

> `trigger.type`이 `signed-tag`가 아니면 `GPG_TOKEN`은 불필요하다.

### Workflow 예시

[example-action.yml](./example-action.yml) 참조. 사용하는 쪽 프로젝트의 `.github/workflows/`에 배치한다.

## 배포 결과물 사용

### GHCR Docker 이미지

```bash
# 로그인
echo $GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin

# pull
docker pull ghcr.io/OWNER/REPO:latest
docker pull ghcr.io/OWNER/REPO:main        # 브랜치명
docker pull ghcr.io/OWNER/REPO:abc1234     # short-sha

# 실행
docker run -p 8080:8080 ghcr.io/OWNER/REPO:latest
```

이미지 태그는 `ci-ghcr.properties`의 `image.name.suffix`에 따라 결정된다.

### GPR Maven 패키지 (Gradle)

`build.gradle.kts`:
```kotlin
repositories {
    maven {
        url = uri("https://maven.pkg.github.com/OWNER/REPO")
        credentials {
            username = System.getenv("GPR_USER")
            password = System.getenv("GPR_TOKEN")
        }
    }
}

dependencies {
    implementation("com.example:artifact:0.0.1-SNAPSHOT")
}
```

### GPR npm 패키지

`.npmrc`:
```
@OWNER:registry=https://npm.pkg.github.com
//npm.pkg.github.com/:_authToken=${GITHUB_TOKEN}
```

```bash
npm install @OWNER/PACKAGE
```

`OWNER/REPO`는 GitHub repository 경로.

### GPR 패키지 접근 토큰

GPR은 public 리포지토리라도 패키지 읽기에 인증이 필요하다. 소비하는 쪽에서 토큰을 발급받아야 한다.

1. GitHub → Settings → Developer settings → **Personal access tokens** → **Fine-grained tokens** → Generate new token
2. 필요 권한:
   - **Repository access**: 패키지가 배포된 리포지토리 선택
   - **Permissions**:
     - `Read access to metadata`
     - `Read access to packages`
3. 발급된 토큰을 환경변수로 설정:
   - **Maven (Gradle)**: `GPR_USER`(GitHub 사용자명), `GPR_TOKEN`(발급 토큰)
   - **npm**: `GITHUB_TOKEN`(발급 토큰) — `.npmrc`에서 참조

### 주의: 패키지 버전 충돌

GPR은 동일 버전의 패키지를 덮어쓸 수 없다. 이미 배포된 버전으로 다시 publish하면 `409 Conflict` 에러가 발생한다.

- **npm**: `package.json`의 `version`을 올려야 한다. 이미 배포된 버전을 삭제하려면 GitHub Settings → Packages에서 수동 삭제.
- **Maven (Gradle)**: `SNAPSHOT` 버전은 덮어쓰기 가능. release 버전은 동일 버전 재배포 불가.

## 개발

외부 의존성 없음 (stdlib only).

```bash
go build -o ci .
go test ./...
go vet ./...
```

### 크로스컴파일

```bash
GOOS=linux  GOARCH=amd64 go build -o ci-linux-amd64 .
GOOS=linux  GOARCH=arm64 go build -o ci-linux-arm64 .
GOOS=darwin GOARCH=amd64 go build -o ci-darwin-amd64 .
GOOS=darwin GOARCH=arm64 go build -o ci-darwin-arm64 .
```
