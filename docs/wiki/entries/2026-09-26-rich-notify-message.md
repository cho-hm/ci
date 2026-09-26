---
title: 웹후크 알림 메시지 확장 및 플랫폼별 꾸밈 (Card 모델, WEBHOOK_LANG)
date: 2026-09-26
issue: -
pr: -
branch: feat/rich-notify
repo: ci
status: open
tags: [notify, webhook, card, i18n, discord, slack, teams, googlechat, telegram, json-v2]
---

## 무엇을 (What)
- 알림에 커밋 메시지/링크, 실행자 프로필, 트리거, 환경, 작업 목록, phase 별 소요 시간, 푸시된 이미지, 실행 번호/재시도, runner, citool 버전 추가.
- 플랫폼별 꾸밈: discord embed, slack attachment, teams adaptive card, googlechat cardsV2, telegram HTML. json 페이로드 v2.
- `WEBHOOK_LANG` (en 기본 | ko).

## 왜 (Why)
- v0.3.0 알림이 텍스트 나열 수준이라 정보·가독성 부족. 사용자가 "같은 정보를 플랫폼마다 지원하는 꾸밈으로" 표현하길 요청.
- 오픈소스라 기본 언어는 영어, 한국어는 옵션.

## 어떻게 (How)
- `Event`(원본 데이터) → `Card`(공통 표현 모델: 제목·facts·commit·phases·artifacts·errors·links·footer, 언어 적용) → 각 `Notifier` 가 Card 를 렌더링. `Notifier.Format(Message{Event, Card})` — json 은 Event 원본을 직접 사용.
- 문구는 `i18n.go` 의 `catalog`(en/ko). 미지원 언어는 경고 후 en 폴백(알림은 계속).
- `PhaseResult` 에 `Elapsed`/`Artifacts` 추가, runner 의 `timed()` 로 측정. build 성공 시 `TaskContexts.ImageRefs()` 를 artifacts 로 보고(도커 빌드 체인도 같은 함수 사용 → 태그 합성 일원화).
- 커밋 메시지는 `GITHUB_EVENT_PATH` 의 `head_commit.message` 첫 줄.
- 버전은 `constant.Version` 을 `-ldflags -X ci/core/constant.Version=<tag>` 로 주입.
- 시작 알림은 플래그 파싱 이후로 이동(작업 목록/환경 표시 위해). 플래그 panic 시엔 시작 없이 실패 알림만.

## 핵심 결정 & 트레이드오프
- Discord 전용 기능(메시지 하나를 수정해 시작→결과로 바꾸기) 제외 — 사용자 결정. 플랫폼 간 동작 일관성 우선.
- Slack 은 Block Kit 대신 legacy attachment — Mattermost/Rocket.Chat 호환 유지.
- Teams/Google Chat 은 스키마가 깊어 map 기반으로 구성, 나머지는 struct.
- HTML 플랫폼은 사용자 입력과 `<1s` 같은 값 이스케이프 필수(Telegram 은 잘못된 HTML 이면 메시지 거부). Google Chat 은 `<code>`·`\n` 미지원 → `<b>`·`<br>`.
- 길이 제한: Discord field 1024, Telegram 4096(초과 시 서식 없는 요약으로 대체).

## 영향 범위
- 파일/모듈: `core/notify/*`(card.go, i18n.go 신규, summary.go 삭제), `core/constant/{result,version}.go`, `core/parse/{model_envs,task_info}.go`, `core/builds/*`, `orch/runner/*`, README, example-action.yml
- 계약 변경: json 페이로드 v1 → v2(필드 추가, 기존 필드 유지). `Notifier` 인터페이스 시그니처 변경(내부).
- 바이너리 빌드: 버전 주입 권장(`~/.claude/scripts/release/go-cross-build.sh` 에 `EXTRA_LDFLAGS`).

## 후속 과제 / 미확정
- 실제 각 플랫폼 렌더링 수동 확인 필요(로컬 수신 서버로 discord/telegram 페이로드만 확인).
- GHES 에서 `<server>/<actor>.png` 프로필 이미지 동작 미확인.

## 링크
- 관련 위키: [[entries/2026-09-26-webhook-notify]]
- 설계 문서/이슈/PR: -
