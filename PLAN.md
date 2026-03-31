# Remote→Local Browser Open Bridge の設計

## 目的

remote 上で動く任意のプロセスが `open browser` を要求したときに、local client 側の daemon がそれを受け取り、**ユーザーの手元のブラウザ**を開ける汎用ブリッジを作る。

対象は Plannotator に限らず、Vite / 各種 dev server / coding agent / docs preview を含む。

---

## 目標

- remote process から local browser open を起動できる
- 特定アプリ依存ではなく、汎用 CLI / daemon として成立する
- SSH/devcontainer でも動く
- 失敗時は単に URL を表示するだけで壊れない
- cloud relay なしで始められる

## 非目標

- 最初から OpenCode / Claude Code / Vite ごとに個別統合すること
- 最初から NAT 越え / SaaS relay / multi-user 運用まで解くこと
- 最初から UI 管理画面を持つこと

---

## 推奨アーキテクチャ

### コンポーネント

1. **Remote caller**
   - Vite や任意アプリ
   - `BROWSER` や wrapper command 経由で URL open を要求する

2. **Remote bridge CLI**
   - 例: `bob open <url>`
   - open request を正規化して local 側へ転送する

3. **Transport adapter**
   - MVP は SSH tunnel / reverse forward 前提
   - 将来 Tailscale / relay を追加可能

4. **Local daemon**
   - user session 上で常駐
   - request を認証・検証して local browser を開く

5. **Local opener**
   - macOS: `open`
   - Linux: `xdg-open`
   - Windows: `start` / ShellExecute

---

## アーキテクチャ図

```text
┌────────────────────┐
│ Remote app/tool    │
│ Vite / agent / etc │
└─────────┬──────────┘
          │ calls
          ▼
┌────────────────────┐
│ Remote bridge CLI  │
│ e.g. bob open URL  │
└─────────┬──────────┘
          │ authenticated request
          ▼
┌────────────────────┐
│ Transport adapter  │
│ SSH / Tailscale    │
│ / relay (later)    │
└─────────┬──────────┘
          │
          ▼
┌────────────────────┐
│ Local daemon       │
│ validate + policy  │
└─────────┬──────────┘
          │ open URL
          ▼
┌────────────────────┐
│ Local browser      │
└────────────────────┘
```

### MVP の最小図

```text
Remote tool -> browser-open-bridge CLI -> SSH tunnel -> local daemon -> OS browser
```

---

## 呼び出しモデル

### 方式A: `BROWSER` を bridge command にする

```bash
export BROWSER=bob
```

ツールが `BROWSER <url>` を尊重するならそのまま使える。

### 方式B: wrapper command を直接呼ぶ

```bash
bob open http://127.0.0.1:5173
```

こちらの方が明示的で安全。

### MVP 推奨

- まずは **wrapper command 直呼び** を正とする
- `BROWSER` 対応は後で足す

---

## 通信プロトコル

### local daemon API

`POST /open`

例:

```json
{
  "version": 1,
  "action": "open_url",
  "url": "http://127.0.0.1:5173",
  "source": {
    "app": "vite",
    "host": "devbox",
    "cwd": "/workspace/app"
  },
  "timestamp": 1712345678,
  "nonce": "random-id"
}
```

### レスポンス

```json
{
  "ok": true,
  "status": "OPENED"
}
```

失敗例:

```json
{
  "ok": false,
  "status": "UNREACHABLE",
  "message": "Local daemon unavailable"
}
```

---

## MVP の transport 方針

### 第一候補: SSH tunnel

理由:
- 追加インフラ不要
- remote dev / SSH / devcontainer と相性が良い
- trust boundary を SSH に寄せられる

### 将来候補

- **Tailscale/private mesh**
  - 個人開発マシンではかなり現実的
- **relay service**
  - 最後の手段。MVPではやらない

---

## セキュリティ方針

### 必須

- `http` / `https` のみ許可
- bearer token で認証
- daemon は privileged にしない
- localhost bind か private network bind のみ
- request log を残す

### 追加で欲しいもの

- host 単位の許可リスト
- external URL を禁止する `localhost-only` mode
- first-use approval
- timestamp + nonce で replay 抑止

---

## 失敗時の挙動

remote 側 CLI は以下の順で degrade する:

1. daemon へ送信を試す
2. 失敗したら URL を標準エラーへ表示
3. アプリ自体は失敗させない

例:

```text
Could not open local browser automatically.
Open this URL on your local machine:
http://127.0.0.1:5173
```

---

## 実装ステップ

### Phase 1: MVP

1. `browser-open-bridge` CLI を作る
2. local daemon を作る
3. `POST /open` API を実装
4. bearer token 認証を入れる
5. SSH tunnel 前提で接続する
6. 失敗時 fallback message を整える

### Phase 2: ergonomics

1. `BROWSER=bob` 対応
2. duplicate open request の抑止
3. log / recent history
4. desktop notification / approval mode

### Phase 3: advanced

1. Tailscale transport
2. relay transport
3. richer actions (`copy`, `focus existing tab`, `open path`)

---

## MVP の判断

この問題には **local daemon + remote CLI + SSH transport** が最も釣り合っている。

理由:
- 特定アプリ依存がない
- OpenCode の UI 制約に依存しない
- Vite など他のワークフローにも効く
- cloud relay なしで始められる

## 最終提案

まずは次の形で進める:

```text
Remote app -> bob CLI -> SSH tunnel -> bobd local daemon -> local browser
```

これを土台にして、必要なら後から `BROWSER` 統合や Tailscale を足す。
