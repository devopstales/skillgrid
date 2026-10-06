# Findings — Local Ollama model catalog

Sourced 2026-10-03 from the Ollama library pages. Tags below are the pull names. Sizes are the library’s published figures, not a local measurement.

| Tag | Kind | Library note | Source |
|-----|------|----------------|--------|
| `qwen2.5:1.5b` | chat, tools | 986MB, 32K, text | https://ollama.com/library/qwen2.5/tags |
| `tev1:0.8b` | decision | 797–812MB, 256K. Together AI, fine-tune of Qwen3.5-0.8B. Requires Ollama 0.35+. Called through `POST /v1/systemone` (`state` + typed `questions`), not chat completions. | https://ollama.com/library/tev1 |
| `embeddinggemma:300m` | embedding | 622MB, 2K. Google EmbeddingGemma, 300M. Requires Ollama 0.11.10+. `POST /api/embed`. `embeddinggemma:latest` is the same 300M weights. | https://ollama.com/library/embeddinggemma |
| `gemma2:2b` | chat | 1.6GB, 8K, text. Library quickstart is `ollama run gemma2:2b` (interactive). | https://ollama.com/library/gemma2 |
| `clef-flash` | decision, vision | 9B, Cloudflare, fine-tune of Qwen3.5-9B. Requires Ollama 0.35.1+. `POST /v1/systemone`. Can score images with the text state. `clef-flash:latest` and `clef-flash:9b` are the tags; there is no smaller size. | https://ollama.com/library/clef-flash |
| `llama3.2:1b` | chat | 1.3GB, 128K, text. Sibling of `llama3.2:3b` (2.0GB), which install pulls today. | https://ollama.com/library/llama3.2 |

**Removed (round 3, 2026-10-06):** `glm:vision-tools` was a typo — not an Ollama library tag. It is dropped from the catalog entirely (no pull entry, no list entry, no warning). The closest real vision+tools GLM tag is `glm-ocr`, but it is NOT a substitute and is NOT pulled. Source: https://ollama.com/library/glm-ocr (checked 2026-10-03).

Version floor for the whole catalog is **Ollama 0.35.1** (clef-flash). Tev1 needs 0.35. EmbeddingGemma needs 0.11.10.

**Locked (round 3):** Live chat model is `llama3.2:1b`. Embedder is `embeddinggemma:300m` (round 2).

`ollama run` starts a REPL. An install step must `ollama pull` and then smoke-test with a non-interactive HTTP call.

## These apps do not call the local tags

Checked 2026-10-03 by cloning `main` and searching for `tev1`, `clef-flash`, and `clef`. Zero matches in all four trees. Each one posts to TypeSafe’s hosted System One API. Pulling the local tags does not make these apps use them.

| Repo | What it calls | Local hook that exists |
|------|----------------|------------------------|
| [compact-adviser](https://github.com/kunchenguid/compact-adviser) | `POST https://api.typesafe.ai/v1/systemone` with `model: "jev-latest"` (fixtures use `jev-1.13.0`). Two yes/no questions per checkpoint. | `TYPESAFE_BASE` may be `http://127.0.0.1` or `localhost`, then `<base>/v1/systemone`. It does not set the model to `tev1` or `clef-flash`. |
| [winnow](https://github.com/GhalebDweikat/winnow) | Default judge `typesafe`, model `jev-latest` (`WINNOW_MODEL`). Fallback is `system-one-adapter` on `claude-haiku-4-5`, not calibrated. | An open-judge experiment trains [jevlike](https://github.com/vinnylarouge/jevlike) on Qwen2.5-0.5B. That is not `tev1:0.8b`. `WINNOW_MODEL` is the only model knob. |
| [jev-review](https://github.com/devagrawal09/jev-review) | `new TypeSafeClient()` from `@typesafe-ai/sdk`, then `client.systemOne({ state, questions })`. No model string in the repo. Needs `TYPESAFE_API_KEY`. | None. No Ollama base URL. |
| [jev-shield](https://github.com/caiovicentino/jev-shield) | Vercel AI SDK `experimental_evaluate` with `model: "typesafe-ai/jev"` in `config/policies.json`. Needs `AI_GATEWAY_API_KEY`. | The README says the model layer can be swapped. The shipped config does not name `tev1` or `clef-flash`. |

`tev1:0.8b` and `clef-flash` speak the same `/v1/systemone` shape these apps already use. Pointing compact-adviser’s `TYPESAFE_BASE` or winnow’s `WINNOW_MODEL` at a local tag would be a separate change. It is not what these repos do today.
