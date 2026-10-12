#!/usr/bin/env node
// `pi` re-enters the skill-pi launcher so a bare `pi` runs with skill-pi's
// packaged extensions — mirroring omegon-pi's bin/pi.mjs legacy shim.
import("./skill-pi.js");
