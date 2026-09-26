---
title: CI 시작/종료 웹후크 알림 (core/notify)
date: 2026-09-26
issue: -
pr: https://github.com/cho-hm/ci/pull/8
branch: feat/webhook-notify
repo: ci
status: merged
tags: [notify, webhook, discord, slack, teams, googlechat, telegram, json, runner, secret]
---

## 무엇을 (What)
- CI 시작 시 / 종료 시(성공·실패 + phase 별 결과 + 소요시간 + 실행 링크) 웹후크 알림 전송.
- 지원 형식: `discord`, `slack`(Mattermost/Rocket.Chat 호환), `teams`, `googlechat`, `telegram`, `json`(범용 폴백).
- 설정: `WEBHOOK_URL`(Secret), `WEBHOOK_TYPE`(선택, 미지정 시 URL 호스트로 자동 판별).

## 왜 (Why)
- 요구: Actions 가동 / 성공·실패를 Discord 로 받고 싶음.
- GitHub Actions 에 Discord 네이티브 알림이 없음. Discord `/github` 엔드포인트는 `workflow_run` 을 처리하지 않아 "시작" 알림 불가, 결과 알림도 불확실.
- citool 은 오픈소스이므로 특정 플랫폼(Discord) 전용이 아닌 플랫폼 중립 구조가 필요.

## 어떻게 (How)
- `core/notify`: `Notifier`(형식 전략: `Name`/`Matches`/`Format`) + 레지스트리(`env` 레지스트리와 동일 관례, `init()` 에서 `Register`).
- `Sender` 인터페이스: `noopSender`(설정 없음/잘못됨) · `webhookSender`(net/http, 10s 타임아웃). Null Object 로 호출부 분기 제거.
- `Event` 는 플랫폼 중립 모델, 공통 요약(`Title`/`Lines`)을 각 Notifier 가 재사용.
- `orch/runner/notification.go`: `parse.TaskContext` → `notify.Event` 변환. `Run()` 시작 시 started, 종료 직전 finished, `defer` + `recover` 로 panic 시 실패 알림 후 re-panic.
- stdlib 전용 원칙 유지.

## 핵심 결정 & 트레이드오프
- citool 내장 — 대안: (a) 워크플로우 yml 에 curl step, (b) 별도 `notify.yml`(`workflow_run` 트리거), (c) Discord `/github` 웹후크. 사용자가 모두 거절. 내장 시 phase 별 결과 등 citool 만 아는 정보를 담을 수 있음.
- 환경변수명 `WEBHOOK_URL` — `DISCORD_WEBHOOK_URL` 로 못박지 않음(플랫폼 중립). 형식은 `WEBHOOK_TYPE` 로 분리.
- URL 은 GitHub Secret 으로만 주입 → 리포/로그 비노출. net/http 에러의 URL 은 `redact` 로 제거.
- 범용 형식 이름은 `generic` 이 아닌 `json`. 외부 계약이므로 `version` 필드 포함.
- 알림은 exit code 에 절대 영향 없음(실패는 WARN 로그).
- SKIPPED phase 는 실패 아님(기존 exit code 판정과 동일).

## 영향 범위
- 파일/모듈: `core/notify/*`(신규), `orch/runner/{runner,notification}.go`, `core/parse/model_envs.go`, `README.md`, `example-action.yml`
- 환경변수 추가: `WEBHOOK_URL`, `WEBHOOK_TYPE` 및 runner 제공 `GITHUB_SERVER_URL`, `GITHUB_RUN_ID`, `GITHUB_WORKFLOW` 읽기. 기존 사용자 호환(미설정 시 비활성).

## 후속 과제 / 미확정
- 바이너리 실행 이전 단계(checkout, 다운로드) 실패는 알림 불가.
- check 단계 goroutine 내부 panic 은 main 의 recover 로 잡히지 않음.
- Dooray / 잔디 / 카카오워크 미지원 — Notifier 파일 1개 + `Register` 로 추가 가능.
- 실제 플랫폼(Discord 등) 대상 수동 검증은 미실시(로컬 수신 서버로 json 형식만 E2E 확인).

## 링크
- 관련 위키: [[entries/2026-09-26-rich-notify-message]] (후속: 메시지 확장·꾸밈)
- 설계 문서/이슈/PR: PR #8, release v0.3.0
