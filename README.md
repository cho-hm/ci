# ci

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)
[![Go](https://img.shields.io/badge/go-1.25-00ADD8?logo=go&logoColor=white)](./go.mod)
[![Platform](https://img.shields.io/badge/platform-linux%20%7C%20darwin%20%7C%20windows-lightgrey)](#개발)

Gradle / Node.js 프로젝트의 **Docker 이미지 빌드(GHCR)** 와 **패키지 배포(GitHub Packages)** 를 단일 Go 바이너리로 처리하는 GitHub Actions CI 도구.

> 외부 의존성 없이 Go stdlib만으로 작성된 단일 실행 파일이며, 사용 측 워크플로우는 release에서 OS/Arch에 맞는 바이너리를 받아 실행합니다.

---

## 특징

- **싱글 바이너리 배포** — release에서 OS/Arch별 바이너리 다운로드 후 즉시 실행. 런타임 의존성 없음.
- **Gradle / Node.js** 환경 모두 지원 — `-env gradle | node` 로 전환.
- **3가지 트리거** — GPG 서명 태그 / 일반 태그 / 브랜치 push.
- **세밀한 이미지 태깅** — `trigger-type`, `tag`, `branch`, `sha`, `short-sha`, `latest` 의 자유로운 조합.
- **부분 실행** — `-publish` / `-build` 를 독립적으로 또는 함께 실행.
- **선택적 설정** — 모든 `.properties` 키는 선택사항이며, 누락 시 안전한 기본값으로 폴백 + 경고 로그.

---

## 목차

- [동작 원리](#동작-원리)
  - [파이프라인](#파이프라인)
  - [Phase 결과 모델](#phase-결과-모델)
- [사전 준비](#사전-준비)
- [Quick Start](#quick-start)
- [실행 플래그](#실행-플래그)
- [트리거 매트릭스](#트리거-매트릭스)
- [프로퍼티 파일](#프로퍼티-파일)
  - [ci-ghcr.properties](#ci-ghcrproperties)
  - [ci-mvn.properties](#ci-mvnproperties)
  - [ci-npm.properties](#ci-npmproperties)
- [환경 변수 & Secrets](#환경-변수--secrets)
- [토큰 발급 가이드](#토큰-발급-가이드)
- [이미지 태그 합성 규칙](#이미지-태그-합성-규칙)
- [기본값 일람](#기본값-일람)
- [시나리오별 예시](#시나리오별-예시)
- [전체 워크플로우 예시](#전체-워크플로우-예시)
- [배포 결과물 소비](#배포-결과물-소비)
  - [GHCR Docker 이미지](#ghcr-docker-이미지)
  - [Maven (Gradle)](#maven-gradle)
  - [npm](#npm)
  - [패키지 버전 충돌 주의](#패키지-버전-충돌-주의)
- [트러블슈팅](#트러블슈팅)
- [개발](#개발)
- [라이선스](#라이선스)

---

## 동작 원리

### 파이프라인

```
┌───────┐    ┌───────┐    ┌─────────────┐
│ parse │ ─▶ │ check │ ─▶ │  publish ?  │
└───────┘    └───────┘ ╲  └─────────────┘
                        ╲ ┌─────────────┐
                         ▶│   build ?   │
                          └─────────────┘
```

| 단계 | 책임 | 주요 동작 |
|------|------|-----------|
| `parse` | 실행 컨텍스트 구성 | 프로퍼티 파일 + GitHub Actions 환경 변수 로드. 파일 누락 시 기본값 폴백 + 경고 로그. |
| `check` | 실행 가부 결정 | 트리거 타입 / 트리거 브랜치 / GPG 서명 검증. 실패 시 후속 phase 를 안전하게 스킵. |
| `publish` | 패키지 배포 | `env=gradle` 이면 `./gradlew publish`, `env=node` 이면 `npm publish`. |
| `build` | Docker 이미지 빌드 & GHCR push | `docker buildx build --push -t ghcr.io/{repo}:{tag1} -t ... {workspace}`. |

`publish` 와 `build` 는 서로 **독립적으로** 실행됩니다. 한쪽이 스킵되거나 실패해도 다른 쪽 실행에는 영향을 주지 않으며, 마지막에 모든 결과를 집계해 **하나라도 실패가 있으면 프로세스는 exit 1** 로 종료됩니다.

### Phase 결과 모델

`check` 의 판단 결과는 후속 phase 의 실행 여부를 결정하고, 최종 결과 라인은 다음 4가지 중 하나로 출력됩니다.

| 결과 | 의미 | 종료코드 영향 |
|------|------|---------------|
| `[publish/build] succeeded` | phase 가 정상 완료 | 영향 없음 |
| `[publish/build] FAILED: ...` | phase 가 실패 (명령 비-제로 종료, 예외 등) | **exit 1** |
| `[publish/build] SKIPPED — not applicable` | 해당 플래그(`-publish`/`-build`) 가 지정되지 않음 | 영향 없음 |
| `[publish/build] SKIPPED — check not passed` | 트리거 조건이 안 맞아 의도된 스킵 | 영향 없음 |
| `[publish/build] SKIPPED — check error` | 검증 중 오류 (예: GPG repo clone 실패) | 영향 없음 |

> 의도된 스킵(`check not passed`)은 정상 종료입니다. 예: 브랜치 push 트리거에서 워크플로우는 동작하되, `trigger.type=signed-tag` 인 phase 는 자연스럽게 스킵됩니다.

---

## 사전 준비

1. **GitHub Repository 권한** — 워크플로우에 다음 권한 선언이 필요합니다.
   ```yaml
   permissions:
     contents: read     # 코드 checkout
     packages: write    # GHCR push, GPR publish
   ```
2. **Dockerfile** (build 사용 시) — 기본 경로는 `<repo>/.github/Dockerfile`. 다른 경로를 쓰려면 `ci-ghcr.properties` 에 `docker.file.path` 로 지정.
3. **Secrets** (필요 시) — `PAT`, `GPG_TOKEN`. 발급 절차는 [토큰 발급 가이드](#토큰-발급-가이드).
4. **GPG 키 저장 리포지토리** (signed-tag 트리거 사용 시) — 별도 private 리포지토리에 `.gpg`/`.asc` 키들을 보관. 자세한 구조는 [ci-ghcr.properties](#ci-ghcrproperties) 참조.

---

## Quick Start

가장 간단한 시작 방법은 [전체 워크플로우 예시](#전체-워크플로우-예시) 의 YAML 을 사용 측 리포의 `.github/workflows/ci.yml` 로 복사하는 것입니다.

이후 다음을 수행하면 끝입니다:

1. 사용할 환경(`gradle` / `node`)에 맞춰 워크플로우의 `CI_ENV` 값을 설정.
2. (선택) 기본값을 덮어쓰고 싶은 키가 있으면 [프로퍼티 파일](#프로퍼티-파일) 을 프로젝트 루트에 작성. **모두 선택사항** 이며, 작성하지 않아도 기본값으로 동작합니다.
3. (선택) 필요한 [secrets](#환경-변수--secrets) 를 리포지토리에 등록.
4. 워크플로우 트리거 조건(태그 push / 브랜치 push) 충족 시 자동 실행.

기본 호출 형태는 다음과 같습니다:

```bash
./ci -env gradle -parse -check -publish -build
```

---

## 실행 플래그

| 플래그 | 포함 단계 | 설명 |
|--------|-----------|------|
| `-env` | — | 빌드 환경. `gradle` (기본) \| `node` |
| `-parse` | parse | 프로퍼티 / 환경 변수만 로드 (디버깅용) |
| `-check` | parse + check | 트리거 조건 검증까지 |
| `-publish` | parse + check + publish | GitHub Packages publish |
| `-build` | parse + check + build | Docker 빌드 + GHCR push |

`-publish` 와 `-build` 는 동시 지정 가능합니다. 어떤 플래그도 없으면 즉시 panic 합니다.

---

## 트리거 매트릭스

`check` 단계는 GitHub Actions 가 워크플로우를 실행한 ref 를 보고, 각 phase 의 `trigger.type` 설정과 비교하여 통과/스킵을 결정합니다.

| 워크플로우 발동 ref | `trigger.type=signed-tag` | `trigger.type=tag` | `trigger.type=branch` |
|---------------------|:--:|:--:|:--:|
| 태그 push (`refs/tags/v1.0.0`) | ✅ GPG 서명 검증 + `trigger.tag` 매칭 시 통과 | ✅ `trigger.tag` 매칭 시 통과 | ❌ 스킵 |
| 브랜치 push (`refs/heads/master`) | ❌ 스킵 | ❌ 스킵 | ✅ `trigger.branch` 매칭 시 통과 |

- `trigger.branch` 는 콜론(`:`)으로 여러 브랜치 지정 가능 — 예: `master:develop:release/*` (단순 동등 비교).
- `trigger.tag` 도 콜론(`:`)으로 여러 태그 지정 가능 — 예: `prod/deploy:stag/deploy:dev/deploy` (단순 동등 비교). **비워두면 모든 태그 통과**(기본값). 매칭 실패 시 `trigger.branch` 와 동일하게 **조용히 스킵**(정상 종료).
- `trigger.tag` 는 `signed-tag` / `tag` 두 타입 모두에 동일하게 적용됩니다 — 서명 여부와 무관하게 "어떤 태그명을 통과시킬지" 하나의 필터로 표현합니다.
- `signed-tag` 는 일반 태그가 아닌 **annotated tag + GPG 서명** 만 통과합니다 (아래 GPG 절차 참고).

**GPG 검증 흐름** (`signed-tag` 트리거 시):

1. `gpg.repo.url` 의 GPG 키 저장 리포를 `git clone --depth=1 --branch <gpg.repo.branch>` 로 임시 디렉터리에 clone.
   - `GPG_TOKEN` 이 설정된 경우 `https://<token>@<host>/<owner>/<repo>` 형태로 인증.
2. `gpg.repo.gpg.path` (기본 `keys/gpg`) / `gpg.repo.asc.path` (기본 `keys/asc`) 하위의 키 파일을 `gpg --batch --import` 로 임포트.
3. `git verify-tag -v refs/tags/<tagName>` 으로 검증.
4. 실패 시 phase 결과는 `SKIPPED — check error`.

---

## 프로퍼티 파일

프로젝트 루트에 배치합니다. **모든 키는 선택사항** — 파일이 없거나 일부 키가 누락되면 안전한 기본값이 사용되며, 누락 시 다음과 같은 경고 로그가 출력됩니다.

```
build properties file not found, using defaults: <path>/ci-ghcr.properties
```

| 파일 | 용도 | 적용 시점 |
|------|------|-----------|
| [`ci-ghcr.properties`](#ci-ghcrproperties) | Docker 빌드 + GHCR push 설정 | `-build` |
| [`ci-mvn.properties`](#ci-mvnproperties)  | Gradle Maven publish 설정 | `-publish` + `env=gradle` |
| [`ci-npm.properties`](#ci-npmproperties)  | npm publish 설정 | `-publish` + `env=node` |

> `-build` 만 사용하면 `ci-mvn.properties`/`ci-npm.properties` 는 아예 읽지 않습니다 (그 반대도 마찬가지).

### ci-ghcr.properties

```properties
# 트리거 타입: signed-tag (GPG 서명 태그) | tag (일반 태그) | branch (브랜치 push)
# 기본값: signed-tag
trigger.type=signed-tag

# 트리거 대상 브랜치. ':'로 여러 브랜치 지정 가능.
# trigger.type=branch 일 때만 사용. 기본값: master
trigger.branch=master:develop

# 트리거 대상 태그. ':'로 여러 태그 지정 가능 (단순 동등 비교).
# trigger.type=signed-tag 또는 tag 일 때만 사용. 기본값: (빈 값 = 모든 태그 통과)
# 매칭 실패 시 trigger.branch 와 동일하게 조용히 스킵(정상 종료).
trigger.tag=prod/deploy:stag/deploy:dev/deploy

# Dockerfile 경로 (워크스페이스 루트 기준)
# 기본값: ./.github/Dockerfile
docker.file.path=./.github/Dockerfile

# 빌드 명령어 — bash -lc 로 실행되므로 '&&', 파이프 등 셸 기능 사용 가능
# 기본값:
#   gradle → ./gradlew clean test bootJar --no-daemon --refresh-dependencies -i
#   node   → npm ci && npm run build
build.command=./gradlew clean test bootJar --no-daemon

# Docker 이미지 멀티-아키텍처 빌드 플랫폼 (쉼표 구분)
# 기본값: linux/amd64,linux/arm64
image.platform=linux/amd64,linux/arm64

# 이미지 태그 suffix 조합 (':'로 구분)
# 사용 가능 토큰: trigger-type, tag, branch, sha, short-sha, latest
# 기본값:
#   gradle → trigger-type:tag:branch:sha:short-sha:latest
#   node   → trigger-type:tag:branch:short-sha:latest
# 자세한 합성 규칙: "이미지 태그 합성 규칙" 섹션 참조
image.name.suffix=trigger-type:tag:branch:sha:short-sha:latest

# ── GPG (signed-tag 트리거 시에만 사용. 그 외엔 무시됨. 기본값 없음) ──

# GPG 키 저장 리포지토리 URL
gpg.repo.url=https://github.com/org/gpg-keys.git

# 리포지토리 내 .gpg 키 디렉터리 경로 (기본값: keys/gpg)
gpg.repo.gpg.path=keys/gpg

# 리포지토리 내 .asc 키 디렉터리 경로 (기본값: keys/asc)
gpg.repo.asc.path=keys/asc

# GPG 리포 clone 시 사용할 브랜치 (기본값: master)
gpg.repo.branch=master
```

**GPG 키 저장 리포지토리 구조 예시**

```
your-org/gpg-keys
├── keys/
│   ├── gpg/
│   │   ├── alice.gpg
│   │   └── bob.gpg
│   └── asc/
│       ├── alice.asc
│       └── bob.asc
```

해당 디렉터리의 모든 파일이 `gpg --batch --import` 로 일괄 임포트됩니다.

### ci-mvn.properties

```properties
# 트리거 타입: signed-tag | tag | branch (기본값: signed-tag)
trigger.type=signed-tag

# 트리거 대상 브랜치. ':'로 여러 브랜치 지정 가능.
# trigger.type=branch 일 때만 사용 (기본값: master)
trigger.branch=master

# 트리거 대상 태그. ':'로 여러 태그 지정 가능 (단순 동등 비교).
# trigger.type=signed-tag 또는 tag 일 때만 사용. 기본값: (빈 값 = 모든 태그 통과)
trigger.tag=prod/deploy:stag/deploy:dev/deploy

# publish 명령어 — sh -c 로 실행. GPR_USER, GPR_TOKEN 환경 변수가 자동 주입됨.
# 기본값: ./gradlew clean test publish --no-daemon
publish.command=./gradlew clean test publish --no-daemon
```

> Gradle 의 publish 단계는 `GPR_USER=$GITHUB_ACTOR`, `GPR_TOKEN=$GITHUB_TOKEN` 두 환경 변수가 자동으로 주입된 상태로 실행됩니다. `build.gradle.kts` 의 `credentials` 블록에서 이를 참조하면 됩니다.

### ci-npm.properties

```properties
# 트리거 타입: signed-tag | tag | branch (기본값: signed-tag)
trigger.type=signed-tag

# 트리거 대상 브랜치. ':'로 여러 브랜치 지정 가능.
# trigger.type=branch 일 때만 사용 (기본값: master)
trigger.branch=master

# 트리거 대상 태그. ':'로 여러 태그 지정 가능 (단순 동등 비교).
# trigger.type=signed-tag 또는 tag 일 때만 사용. 기본값: (빈 값 = 모든 태그 통과)
trigger.tag=prod/deploy:stag/deploy:dev/deploy

# publish 명령어 — sh -c 로 실행
# 기본값: npm publish --registry=https://npm.pkg.github.com
publish.command=npm publish --registry=https://npm.pkg.github.com
```

> npm publish 단계는 자동으로 워크스페이스 루트에 `.npmrc` 를 생성합니다 (`@<owner>:registry=...`, `_authToken=$GITHUB_TOKEN`). 별도 환경 변수 주입은 필요 없습니다.

---

## 환경 변수 & Secrets

워크플로우의 `env` 블록을 통해 바이너리에 전달합니다.

| 변수 | 필수 | 설명 |
|------|:----:|------|
| `GITHUB_TOKEN` | ✅ | `${{ secrets.GITHUB_TOKEN }}` — GitHub Actions 가 자동 발급 |
| `PAT` | ◻️ | Personal Access Token. GHCR 로그인용. 미설정 시 `GITHUB_TOKEN` 으로 폴백 |
| `GPG_TOKEN` | ◻️ | `signed-tag` 트리거 시 GPG 키 저장 리포 clone 용 인증 토큰 |

다음 변수들은 GitHub Actions runner 가 자동 주입하므로 별도 설정이 필요 없습니다.

`GITHUB_WORKSPACE`, `GITHUB_REPOSITORY`, `GITHUB_REF_TYPE`, `GITHUB_REF`, `GITHUB_REF_NAME`, `GITHUB_SHA`, `GITHUB_ACTOR`

---

## 토큰 발급 가이드

모든 토큰은 GitHub → **Settings → Developer settings → Personal access tokens → Fine-grained tokens → Generate new token** 에서 발급한 후, 사용할 리포지토리의 **Settings → Secrets and variables → Actions → New repository secret** 에 등록합니다.

| 토큰 | 발급 시점 | Repository access | Permissions |
|------|-----------|-------------------|-------------|
| **`PAT`** | GHCR 권한이 별도 필요할 때 (선택) | 대상 리포지토리 | `metadata: read`, `packages: read & write` |
| **`GPG_TOKEN`** | `trigger.type=signed-tag` 사용 시 | GPG 키 저장 리포지토리 | `metadata: read`, `contents: read` |
| **GPR 소비자 토큰** | 배포된 패키지를 사용하는 측 (별도 등록) | 패키지 배포 리포지토리 | `metadata: read`, `packages: read` |

> `GITHUB_TOKEN` 은 workflow 가 자동 발급하므로 별도 작업이 필요 없습니다 — 워크플로우의 `permissions` 만 선언하면 됩니다.

---

## 이미지 태그 합성 규칙

`image.name.suffix` 는 콜론(`:`)으로 구분된 토큰 시퀀스이며, 각 토큰이 활성화될 때 다음 값이 태그로 추가됩니다.

| 토큰 | 추가되는 값 | 조건 |
|------|-------------|------|
| `trigger-type` | `trigger.type` 값 그대로 (`signed-tag` / `tag` / `branch`) | 항상 |
| `tag` | `GITHUB_REF_NAME` (예: `v1.0.0`) | trigger 가 tag 계열일 때만 |
| `branch` | `GITHUB_REF_NAME` 그대로 | 항상 (이름 그대로 추가됨) |
| `sha` | 전체 commit SHA (40자) | 항상 |
| `short-sha` | commit SHA 앞 7자 | 항상 |
| `latest` | 문자열 `"latest"` | 항상 |

생성된 태그들은 **`docker buildx build --push -t ghcr.io/<repo>:<tag1> -t ghcr.io/<repo>:<tag2> ... <workspace>`** 로 한 번에 push 됩니다. (`<repo>` 는 소유자/이름이 모두 소문자 변환됩니다.)

**예시 1 — signed-tag 트리거**

- 설정: `image.name.suffix=trigger-type:tag:branch:short-sha:latest`, `trigger.type=signed-tag`
- 워크플로우 발동: `git push --tags v1.0.0` (커밋 sha = `abc123def456...`)
- 결과 태그:
  - `ghcr.io/acme/widget:signed-tag`
  - `ghcr.io/acme/widget:v1.0.0`
  - `ghcr.io/acme/widget:abc123d`
  - `ghcr.io/acme/widget:latest`

**예시 2 — branch 트리거**

- 설정: `image.name.suffix=branch:short-sha:latest`, `trigger.type=branch`, `trigger.branch=master`
- 워크플로우 발동: master 브랜치 push (sha = `def456abc789...`)
- 결과 태그:
  - `ghcr.io/acme/widget:master`
  - `ghcr.io/acme/widget:def456a`
  - `ghcr.io/acme/widget:latest`

---

## 기본값 일람

`-env` 값에 따라 일부 기본값이 다릅니다.

| 키 | Gradle (`-env gradle`) | Node (`-env node`) |
|----|------------------------|---------------------|
| `build.command` | `./gradlew clean test bootJar --no-daemon --refresh-dependencies -i` | `npm ci && npm run build` |
| `publish.command` | `./gradlew clean test publish --no-daemon` | `npm publish --registry=https://npm.pkg.github.com` |
| `image.name.suffix` | `trigger-type:tag:branch:sha:short-sha:latest` | `trigger-type:tag:branch:short-sha:latest` |
| `docker.file.path` | `./.github/Dockerfile` | `./.github/Dockerfile` |
| `image.platform` | `linux/amd64,linux/arm64` | `linux/amd64,linux/arm64` |
| publish properties 파일 | `ci-mvn.properties` | `ci-npm.properties` |

`-env` 와 무관한 공통 기본값:

| 키 | 기본값 |
|----|--------|
| `trigger.type` | `signed-tag` |
| `trigger.branch` | `master` |
| `trigger.tag` | (빈 값 — 모든 태그 통과) |
| `gpg.repo.gpg.path` | `keys/gpg` |
| `gpg.repo.asc.path` | `keys/asc` |
| `gpg.repo.branch` | `master` |
| `gpg.repo.url` | (없음) |

---

## 시나리오별 예시

### 시나리오 A — Gradle + signed-tag 릴리즈 빌드

- **목표**: GPG 서명된 태그 push 시에만 Maven publish + Docker 이미지 빌드
- 워크플로우 트리거: `tags: ["*"]`
- `ci-mvn.properties`: 작성 안 함 (모두 기본값 사용)
- `ci-ghcr.properties`:
  ```properties
  trigger.type=signed-tag
  gpg.repo.url=https://github.com/your-org/gpg-keys.git
  ```
- 필요한 secrets: `GPG_TOKEN`
- 실행 플래그: `-env gradle -publish -build`

### 시나리오 B — Node + branch 트리거 (개발 빌드)

- **목표**: develop 브랜치 push 마다 Docker 이미지만 빌드 (publish 안 함)
- 워크플로우 트리거: `branches: [develop]`
- `ci-ghcr.properties`:
  ```properties
  trigger.type=branch
  trigger.branch=develop
  image.name.suffix=branch:short-sha:latest
  ```
- 실행 플래그: `-env node -build`

### 시나리오 C — Node + tag 트리거 + 특정 태그만 빌드

- **목표**: 환경별 배포 태그(`prod/deploy`, `stag/deploy`, `dev/deploy`)에서만 Docker 이미지 빌드. 다른 태그(예: `nightly-*`)는 조용히 스킵.
- 워크플로우 트리거: `tags: ["*"]`
- `ci-ghcr.properties`:
  ```properties
  trigger.type=tag
  trigger.tag=prod/deploy:stag/deploy:dev/deploy
  ```
- 실행 플래그: `-env node -build`

> `trigger.tag` 가 비어있으면 모든 태그가 통과하므로, 특정 태그 화이트리스트가 필요한 경우에만 지정합니다. 매칭 실패는 정상 스킵으로 취급됩니다 (exit 0).

### 시나리오 D — Gradle + branch 트리거 + 커스텀 빌드 명령

- **목표**: master push 시 통합 테스트 포함 빌드 후 GHCR push
- `ci-ghcr.properties`:
  ```properties
  trigger.type=branch
  trigger.branch=master
  build.command=./gradlew clean integrationTest bootJar --no-daemon
  image.platform=linux/amd64
  ```
- 실행 플래그: `-env gradle -build`

---

## 전체 워크플로우 예시

다음 YAML 을 사용 측 리포의 `.github/workflows/ci.yml` 으로 복사하면 즉시 동작합니다. `CI_VERSION` 은 사용할 `cho-hm/ci` release 태그로 교체하세요.

```yaml
name: CI

on:
  push:
    branches: [master, develop]
    tags: ["*"]

env:
  CI_VERSION: "v0.1.0"  # cho-hm/ci 릴리즈 버전
  CI_ENV: "gradle"      # "gradle" 또는 "node"

jobs:
  ci:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      packages: write

    steps:
      # ──────────────────────────────────────────
      # Pre-steps
      # ──────────────────────────────────────────
      - name: Checkout
        uses: actions/checkout@v4
        with:
          fetch-depth: 0
          fetch-tags: true

      - name: Setup Java
        if: env.CI_ENV == 'gradle'
        uses: actions/setup-java@v4
        with:
          distribution: temurin
          java-version: "21"
          cache: gradle

      - name: Setup Node
        if: env.CI_ENV == 'node'
        uses: actions/setup-node@v4
        with:
          node-version: "20"
          registry-url: https://npm.pkg.github.com

      - name: Setup Docker Buildx
        uses: docker/setup-buildx-action@v3

      # ──────────────────────────────────────────
      # CI 바이너리 다운로드
      # ──────────────────────────────────────────
      - name: Download CI binary
        shell: bash
        run: |
          set -euo pipefail

          case "${{ runner.os }}" in
            Linux)   GOOS="linux" ;;
            macOS)   GOOS="darwin" ;;
            Windows) GOOS="windows" ;;
            *) echo "Unsupported OS: ${{ runner.os }}"; exit 1 ;;
          esac
          case "${{ runner.arch }}" in
            X64)   GOARCH="amd64" ;;
            ARM64) GOARCH="arm64" ;;
            *) echo "Unsupported ARCH: ${{ runner.arch }}"; exit 1 ;;
          esac

          NAME="ci-${GOOS}-${GOARCH}"
          EXT=""; if [ "$GOOS" = "windows" ]; then EXT=".exe"; fi

          URL_PREFIX="https://github.com/cho-hm/ci/releases/download/${{ env.CI_VERSION }}"
          OUT_DIR="${{ runner.temp }}/${NAME}"
          mkdir -p "$OUT_DIR"
          BIN_PATH="$OUT_DIR/${NAME}${EXT}"

          curl -fL --retry 3 --retry-delay 2 -o "$BIN_PATH" "${URL_PREFIX}/${NAME}${EXT}"

          if [ "$GOOS" != "windows" ]; then chmod +x "$BIN_PATH"; fi
          echo "CI_PATH=$BIN_PATH" >> "$GITHUB_ENV"

      # ──────────────────────────────────────────
      # CI 실행
      # ──────────────────────────────────────────
      - name: Run CI
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          PAT:          ${{ secrets.PAT }}
          GPG_TOKEN:    ${{ secrets.GPG_TOKEN }}
        run: |
          "$CI_PATH" -env "${{ env.CI_ENV }}" -parse -check -publish -build
```

---

## 배포 결과물 소비

### GHCR Docker 이미지

```bash
# 로그인 (소비자 측)
echo $GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin

# pull
docker pull ghcr.io/OWNER/REPO:latest
docker pull ghcr.io/OWNER/REPO:v1.0.0      # tag
docker pull ghcr.io/OWNER/REPO:master      # 브랜치
docker pull ghcr.io/OWNER/REPO:abc1234     # short-sha

# 실행
docker run -p 8080:8080 ghcr.io/OWNER/REPO:latest
```

생성되는 태그 종류는 `image.name.suffix` 에 따라 달라집니다 — [이미지 태그 합성 규칙](#이미지-태그-합성-규칙) 참조.

### Maven (Gradle)

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

소비자 측 환경 변수: `GPR_USER` (GitHub 사용자명), `GPR_TOKEN` ([GPR 소비자 토큰](#토큰-발급-가이드)).

### npm

`.npmrc`:

```
@OWNER:registry=https://npm.pkg.github.com
//npm.pkg.github.com/:_authToken=${GITHUB_TOKEN}
```

```bash
npm install @OWNER/PACKAGE
```

소비자 측 환경 변수: `GITHUB_TOKEN` (실제 값은 [GPR 소비자 토큰](#토큰-발급-가이드)).

> `OWNER/REPO` 는 GitHub 리포지토리 경로입니다.

### 패키지 버전 충돌 주의

GitHub Packages 는 동일 버전 패키지의 덮어쓰기를 제한합니다. 이미 배포된 버전으로 다시 publish하면 `409 Conflict` 가 발생합니다.

- **npm** — `package.json` 의 `version` 을 올려야 합니다. 배포된 버전 삭제는 GitHub → Settings → Packages 에서 수동으로 가능합니다.
- **Maven (Gradle)** — `SNAPSHOT` 버전은 덮어쓰기 가능하지만, release 버전은 동일 버전 재배포가 불가합니다.

---

## 트러블슈팅

| 증상 | 원인 / 확인 사항 |
|------|------------------|
| `[build] SKIPPED — check not passed` (브랜치 push 시) | `trigger.type=signed-tag` 인데 태그가 아닌 브랜치를 push 한 경우. 의도된 스킵입니다. |
| `[build] SKIPPED — check not passed` + `Expected trigger tag=...` 로그 | `trigger.tag` 에 명시된 태그 목록과 push 된 태그 이름이 일치하지 않음. `trigger.branch` 와 동일한 의도된 스킵입니다. |
| `[build] SKIPPED — check error` | `gpg.repo.url` clone 실패, `git verify-tag` 실패 등. 워크플로우 로그의 직전 줄을 확인하세요. |
| `Expected trigger type=signed-tag, but actual: branch` | 동일 — phase 의 `trigger.type` 과 실제 ref 가 일치하지 않을 때 표시되는 정상 로그입니다. |
| `gpg signature verification failed for tag <name>` | 해당 태그가 GPG 서명되지 않았거나 사용된 키가 `gpg.repo.url` 에 없음. `git tag -v <name>` 으로 로컬 검증해 보세요. |
| `docker login` 실패 | `PAT` 권한 부족 — `packages: read & write` 가 있는지 확인. `PAT` 미등록 시 `GITHUB_TOKEN` 으로 시도하지만, 워크플로우의 `permissions: packages: write` 선언이 누락되었을 수 있음. |
| `409 Conflict` (publish) | 동일 버전 재배포. 버전을 올리거나 SNAPSHOT 사용. |
| `properties file not found, using defaults` | 정상 동작 — 해당 phase 가 적용되지만 properties 파일이 없는 경우의 안내 로그. 의도된 것이라면 무시해도 됩니다. |
| `Need some options, but nothing` panic | 실행 플래그가 하나도 없음. `-parse`/`-check`/`-publish`/`-build` 중 하나 이상 지정. |

---

## 개발

### 빌드 & 테스트

외부 의존성이 없는 stdlib-only 프로젝트입니다 (Go 1.25).

```bash
go build -o ci .
go test ./...
go vet ./...
```

### 크로스 컴파일

릴리즈 바이너리는 다음 6종을 권장합니다.

```bash
GOOS=linux   GOARCH=amd64 go build -o ci-linux-amd64 .
GOOS=linux   GOARCH=arm64 go build -o ci-linux-arm64 .
GOOS=darwin  GOARCH=amd64 go build -o ci-darwin-amd64 .
GOOS=darwin  GOARCH=arm64 go build -o ci-darwin-arm64 .
GOOS=windows GOARCH=amd64 go build -o ci-windows-amd64.exe .
GOOS=windows GOARCH=arm64 go build -o ci-windows-arm64.exe .
```

GitHub Releases 에 업로드하면 [전체 워크플로우 예시](#전체-워크플로우-예시) 의 다운로드 단계가 OS/Arch 에 맞춰 자동 선택합니다.

### 디렉터리 구조

```
.
├── main.go                    # 진입점 — runner.Run() 호출만
├── orch/                      # orchestration 계층
│   ├── arg/                   # CLI 플래그 파싱
│   └── runner/                # 단계 조율 (parse → check → publish/build)
├── core/                      # 도메인 계층
│   ├── parse/                 # 프로퍼티 파일 + 환경 변수 로드
│   ├── check/                 # 트리거 검증 + GPG 서명 검증
│   ├── builds/                # Docker 빌드 체인 (build → login → buildx → logout)
│   ├── publish/               # Maven/npm publish
│   ├── env/                   # gradle/node 환경별 기본값/체인
│   └── constant/              # 전역 상수, phase 결과 모델
└── util/
    ├── cli/                   # exec.Command 래퍼
    └── git/                   # git 명령 래퍼
```

---

## 라이선스

이 프로젝트는 [MIT License](./LICENSE) 하에 배포됩니다.
