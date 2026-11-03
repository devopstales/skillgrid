"""Mnemonic MemoryProvider plugin for Hermes Agent (TICKET-01).

Entry point discovered by Hermes plugin loading: `register(ctx)` hands the
provider instance to the Hermes runtime, which drives its memory lifecycle
(prefetch, sync_turn, system prompt block, tool schemas, tool calls).
"""

from .provider import MnemonicMemoryProvider


def register(ctx):
    ctx.register_memory_provider(MnemonicMemoryProvider())
