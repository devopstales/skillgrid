# Review package: 7cc50f4..76ec335

## Commits
76ec335 feat(ui): decisions view with answer round-trip (TICKET-04)
b7e39a4 test(ui): RED acceptance tests for decisions view (TICKET-03)

## Files changed
 skillgrid-ui/package-lock.json                     | 681 +++++++++++++++++++++
 skillgrid-ui/package.json                          |   3 +
 skillgrid-ui/src/app.tsx                           |  14 +
 skillgrid-ui/src/components/layout/AppLayout.tsx   |   2 +
 .../src/features/decisions/DecisionCard.test.tsx   |  65 ++
 .../src/features/decisions/DecisionCard.tsx        | 165 +++++
 .../src/features/decisions/DecisionsPage.test.tsx  |  84 +++
 .../src/features/decisions/DecisionsPage.tsx       |  86 +++
 skillgrid-ui/src/features/decisions/api.ts         |  78 +++
 skillgrid-ui/src/vite-env.d.ts                     |   1 +
 skillgrid-ui/vite.config.ts                        |  10 +
 11 files changed, 1189 insertions(+)

## Diff
diff --git a/skillgrid-ui/package-lock.json b/skillgrid-ui/package-lock.json
index 0fd2aae..026776b 100644
--- a/skillgrid-ui/package-lock.json
+++ b/skillgrid-ui/package-lock.json
@@ -22,42 +22,98 @@
         "mermaid": "^12.0.0",
         "react": "^19.1.0",
         "react-dom": "^19.1.0",
         "react-markdown": "^10.1.0",
         "rehype-sanitize": "^6.0.0",
         "remark-gfm": "^4.0.1",
         "sigma": "^3.0.3"
       },
       "devDependencies": {
         "@tailwindcss/vite": "^4.1.11",
+        "@testing-library/dom": "^10.4.2",
+        "@testing-library/react": "^16.3.3",
         "@types/react": "^19.1.8",
         "@types/react-dom": "^19.1.6",
         "@vitejs/plugin-react": "^4.6.0",
+        "jsdom": "^30.1.0",
         "tailwindcss": "^4.1.11",
         "typescript": "~5.8.3",
         "vite": "^6.3.5",
         "vitest": "^5.0.1"
       }
     },
     "node_modules/@antfu/install-pkg": {
       "version": "2.1.0",
       "resolved": "https://registry.npmjs.org/@antfu/install-pkg/-/install-pkg-2.1.0.tgz",
       "integrity": "sha512-sdg9NxU3zR4Mnawfbc/x6GB5Wf17WYud5qOuEuxXjaKpYpMkISSJEjItGebXJ2bQ4DIcly4NYH23mtkGJjvKUw==",
       "license": "MIT",
       "dependencies": {
         "package-manager-detector": "^1.8.0",
         "tinyexec": "^1.3.1"
       },
       "funding": {
         "url": "https://github.com/sponsors/antfu"
       }
     },
+    "node_modules/@asamuzakjp/css-color": {
+      "version": "7.0.1",
+      "resolved": "https://registry.npmjs.org/@asamuzakjp/css-color/-/css-color-7.0.1.tgz",
+      "integrity": "sha512-C9duntabagkBZ1LebM7FKmphR4Q1pBclLxVbZETQV0akkFjV0ooFxo8FlvAQyKj9F6l8rEnmidgcTlpyxFYizg==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@csstools/css-calc": "^3.4.0",
+        "@csstools/css-color-parser": "^4.2.3",
+        "@csstools/css-parser-algorithms": "^4.0.0",
+        "@csstools/css-tokenizer": "^4.0.1",
+        "lru-cache": "^11.5.3"
+      },
+      "engines": {
+        "node": "^22.22.2 || ^24.15.0 || >=26.0.0"
+      }
+    },
+    "node_modules/@asamuzakjp/css-color/node_modules/lru-cache": {
+      "version": "11.5.3",
+      "resolved": "https://registry.npmjs.org/lru-cache/-/lru-cache-11.5.3.tgz",
+      "integrity": "sha512-U4N8FgzmWxc8k1VH8Kr6lQg18U7Fjvby6wXHVRX/ZZ7IwWbRMgrRbP0Wrb5q5NVinryp4SQampHKdvtecItxUg==",
+      "dev": true,
+      "license": "BlueOak-1.0.0",
+      "engines": {
+        "node": "20 || >=22"
+      }
+    },
+    "node_modules/@asamuzakjp/dom-selector": {
+      "version": "9.2.1",
+      "resolved": "https://registry.npmjs.org/@asamuzakjp/dom-selector/-/dom-selector-9.2.1.tgz",
+      "integrity": "sha512-NT4s3yZLjovPpliRpTvdsdzyPjqRqiCZj9MxnarBihbaY5MbAG7DWAJcrLlhmC7xKTNCXsTELcqB8YsqpLnUFQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "bidi-js": "^1.1.0",
+        "css-tree": "^3.2.1",
+        "is-potential-custom-element-name": "^1.0.1",
+        "lru-cache": "^11.5.3"
+      },
+      "engines": {
+        "node": "^22.22.2 || ^24.15.0 || >=26.0.0"
+      }
+    },
+    "node_modules/@asamuzakjp/dom-selector/node_modules/lru-cache": {
+      "version": "11.5.3",
+      "resolved": "https://registry.npmjs.org/lru-cache/-/lru-cache-11.5.3.tgz",
+      "integrity": "sha512-U4N8FgzmWxc8k1VH8Kr6lQg18U7Fjvby6wXHVRX/ZZ7IwWbRMgrRbP0Wrb5q5NVinryp4SQampHKdvtecItxUg==",
+      "dev": true,
+      "license": "BlueOak-1.0.0",
+      "engines": {
+        "node": "20 || >=22"
+      }
+    },
     "node_modules/@babel/code-frame": {
       "version": "7.29.7",
       "resolved": "https://registry.npmjs.org/@babel/code-frame/-/code-frame-7.29.7.tgz",
       "integrity": "sha512-Aup7aUOfpbAUg2ROOJN6Iw5f9DMBlzu0mIkm/malLQFN/YQgO48wCj0Kxa3sEHJvPVFg7siR+qRInwXd2qhQKw==",
       "dev": true,
       "license": "MIT",
       "dependencies": {
         "@babel/helper-validator-identifier": "^7.29.7",
         "js-tokens": "^4.0.0",
         "picocolors": "^1.1.1"
@@ -278,20 +334,30 @@
       "dependencies": {
         "@babel/helper-plugin-utils": "^7.29.7"
       },
       "engines": {
         "node": ">=6.9.0"
       },
       "peerDependencies": {
         "@babel/core": "^7.0.0-0"
       }
     },
+    "node_modules/@babel/runtime": {
+      "version": "7.29.7",
+      "resolved": "https://registry.npmjs.org/@babel/runtime/-/runtime-7.29.7.tgz",
+      "integrity": "sha512-Nq8OhGWiZIZGV6hLHoyAKLLcJihP/xFeBMGJoUrxTX2psI8dCifzLhZISFb+VWS3wFMRDmCGw5R+dOySCqPLhw==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=6.9.0"
+      }
+    },
     "node_modules/@babel/template": {
       "version": "7.29.7",
       "resolved": "https://registry.npmjs.org/@babel/template/-/template-7.29.7.tgz",
       "integrity": "sha512-puq+Gf35oI24FeN11LkoUQFqv9uwNeWpxXZi/Ji3rRIoKAzKnxRaZ+Gkj0vKS9ZCiTESfng1N9LyOyXvo+m+Gg==",
       "dev": true,
       "license": "MIT",
       "dependencies": {
         "@babel/code-frame": "^7.29.7",
         "@babel/parser": "^7.29.7",
         "@babel/types": "^7.29.7"
@@ -332,20 +398,33 @@
       "engines": {
         "node": ">=6.9.0"
       }
     },
     "node_modules/@braintree/sanitize-url": {
       "version": "7.1.2",
       "resolved": "https://registry.npmjs.org/@braintree/sanitize-url/-/sanitize-url-7.1.2.tgz",
       "integrity": "sha512-jigsZK+sMF/cuiB7sERuo9V7N9jx+dhmHHnQyDSVdpZwVutaBu7WvNYqMDLSgFgfB30n452TP3vjDAvFC973mA==",
       "license": "MIT"
     },
+    "node_modules/@bramus/specificity": {
+      "version": "2.4.2",
+      "resolved": "https://registry.npmjs.org/@bramus/specificity/-/specificity-2.4.2.tgz",
+      "integrity": "sha512-ctxtJ/eA+t+6q2++vj5j7FYX3nRu311q1wfYH3xjlLOsczhlhxAg2FWNUXhpGvAw3BWo1xBcvOV6/YLc2r5FJw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "css-tree": "^3.0.0"
+      },
+      "bin": {
+        "specificity": "bin/cli.js"
+      }
+    },
     "node_modules/@chevrotain/cst-dts-gen": {
       "version": "11.1.2",
       "resolved": "https://registry.npmjs.org/@chevrotain/cst-dts-gen/-/cst-dts-gen-11.1.2.tgz",
       "integrity": "sha512-XTsjvDVB5nDZBQB8o0o/0ozNelQtn2KrUVteIHSlPd2VAV2utEb6JzyCJaJ8tGxACR4RiBNWy5uYUHX2eji88Q==",
       "license": "Apache-2.0",
       "dependencies": {
         "@chevrotain/gast": "11.1.2",
         "@chevrotain/types": "11.1.2",
         "lodash-es": "4.17.23"
       }
@@ -371,20 +450,160 @@
       "resolved": "https://registry.npmjs.org/@chevrotain/types/-/types-11.1.2.tgz",
       "integrity": "sha512-U+HFai5+zmJCkK86QsaJtoITlboZHBqrVketcO2ROv865xfCMSFpELQoz1GkX5GzME8pTa+3kbKrZHQtI0gdbw==",
       "license": "Apache-2.0"
     },
     "node_modules/@chevrotain/utils": {
       "version": "11.1.2",
       "resolved": "https://registry.npmjs.org/@chevrotain/utils/-/utils-11.1.2.tgz",
       "integrity": "sha512-4mudFAQ6H+MqBTfqLmU7G1ZwRzCLfJEooL/fsF6rCX5eePMbGhoy5n4g+G4vlh2muDcsCTJtL+uKbOzWxs5LHA==",
       "license": "Apache-2.0"
     },
+    "node_modules/@csstools/color-helpers": {
+      "version": "6.1.1",
+      "resolved": "https://registry.npmjs.org/@csstools/color-helpers/-/color-helpers-6.1.1.tgz",
+      "integrity": "sha512-gLNsunvwf3mCi5u5o46/Z/JcJMnhbHSaZ69rkgPzNM3J4s8hWwpPUQB6/tt0EDFyCiWzxANlx+2LJwpYj4zS1w==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "github",
+          "url": "https://github.com/sponsors/csstools"
+        },
+        {
+          "type": "opencollective",
+          "url": "https://opencollective.com/csstools"
+        }
+      ],
+      "license": "MIT-0",
+      "engines": {
+        "node": ">=20.19.0"
+      }
+    },
+    "node_modules/@csstools/css-calc": {
+      "version": "3.4.0",
+      "resolved": "https://registry.npmjs.org/@csstools/css-calc/-/css-calc-3.4.0.tgz",
+      "integrity": "sha512-XQKj5B7QiZcHiegCOCAzcAOJdhGgWOHbbu62h5e5mkHnn8lWcfiJhllkqWmxu5zWR9jucPHuo1iTB56P033hcg==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "github",
+          "url": "https://github.com/sponsors/csstools"
+        },
+        {
+          "type": "opencollective",
+          "url": "https://opencollective.com/csstools"
+        }
+      ],
+      "license": "MIT",
+      "engines": {
+        "node": ">=20.19.0"
+      },
+      "peerDependencies": {
+        "@csstools/css-parser-algorithms": "^4.0.0",
+        "@csstools/css-tokenizer": "^4.0.0"
+      }
+    },
+    "node_modules/@csstools/css-color-parser": {
+      "version": "4.2.3",
+      "resolved": "https://registry.npmjs.org/@csstools/css-color-parser/-/css-color-parser-4.2.3.tgz",
+      "integrity": "sha512-y4LpL+lmpuyKDiEFq2PnZUVFdAjsoB/qQJod79yLNokXyW7jewi+/WJ69EfItj8A2unWtxXnGjw6LYXgXu5ZjA==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "github",
+          "url": "https://github.com/sponsors/csstools"
+        },
+        {
+          "type": "opencollective",
+          "url": "https://opencollective.com/csstools"
+        }
+      ],
+      "license": "MIT",
+      "dependencies": {
+        "@csstools/color-helpers": "^6.1.1",
+        "@csstools/css-calc": "^3.4.0"
+      },
+      "engines": {
+        "node": ">=20.19.0"
+      },
+      "peerDependencies": {
+        "@csstools/css-parser-algorithms": "^4.0.0",
+        "@csstools/css-tokenizer": "^4.0.0"
+      }
+    },
+    "node_modules/@csstools/css-parser-algorithms": {
+      "version": "4.0.0",
+      "resolved": "https://registry.npmjs.org/@csstools/css-parser-algorithms/-/css-parser-algorithms-4.0.0.tgz",
+      "integrity": "sha512-+B87qS7fIG3L5h3qwJ/IFbjoVoOe/bpOdh9hAjXbvx0o8ImEmUsGXN0inFOnk2ChCFgqkkGFQ+TpM5rbhkKe4w==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "github",
+          "url": "https://github.com/sponsors/csstools"
+        },
+        {
+          "type": "opencollective",
+          "url": "https://opencollective.com/csstools"
+        }
+      ],
+      "license": "MIT",
+      "engines": {
+        "node": ">=20.19.0"
+      },
+      "peerDependencies": {
+        "@csstools/css-tokenizer": "^4.0.0"
+      }
+    },
+    "node_modules/@csstools/css-syntax-patches-for-csstree": {
+      "version": "1.1.14",
+      "resolved": "https://registry.npmjs.org/@csstools/css-syntax-patches-for-csstree/-/css-syntax-patches-for-csstree-1.1.14.tgz",
+      "integrity": "sha512-HpbVXyrofRXpHpgkNIjU/3EWR4WJvOkO3emNK/L6X/mTJU7bGUI3AkkpoTNXznQLp0KRjLHELTGeKI5dIkI9JQ==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "github",
+          "url": "https://github.com/sponsors/csstools"
+        },
+        {
+          "type": "opencollective",
+          "url": "https://opencollective.com/csstools"
+        }
+      ],
+      "license": "MIT-0",
+      "peerDependencies": {
+        "css-tree": "^3.2.1"
+      },
+      "peerDependenciesMeta": {
+        "css-tree": {
+          "optional": true
+        }
+      }
+    },
+    "node_modules/@csstools/css-tokenizer": {
+      "version": "4.0.1",
+      "resolved": "https://registry.npmjs.org/@csstools/css-tokenizer/-/css-tokenizer-4.0.1.tgz",
+      "integrity": "sha512-bPlN9S9O1A0euCpEWE4qnvB5YDuyYVsUTrxSgmAM1Is0j4tICHoVyOVAXfWMP/kS9ZrjvyIXWV2PmomiAXXqOw==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "github",
+          "url": "https://github.com/sponsors/csstools"
+        },
+        {
+          "type": "opencollective",
+          "url": "https://opencollective.com/csstools"
+        }
+      ],
+      "license": "MIT",
+      "engines": {
+        "node": ">=20.19.0"
+      }
+    },
     "node_modules/@dnd-kit/accessibility": {
       "version": "3.1.1",
       "resolved": "https://registry.npmjs.org/@dnd-kit/accessibility/-/accessibility-3.1.1.tgz",
       "integrity": "sha512-2P+YgaXF+gRsIihwwY1gCsQSYnu9Zyj2py8kY5fFvUM1qm2WA2u639R6YNVfU4GWr+ZM5mqEsfHZZLoRONbemw==",
       "license": "MIT",
       "dependencies": {
         "tslib": "^2.0.0"
       },
       "peerDependencies": {
         "react": ">=16.8.0"
@@ -866,20 +1085,38 @@
       "dev": true,
       "license": "MIT",
       "optional": true,
       "os": [
         "win32"
       ],
       "engines": {
         "node": ">=18"
       }
     },
+    "node_modules/@exodus/bytes": {
+      "version": "1.15.1",
+      "resolved": "https://registry.npmjs.org/@exodus/bytes/-/bytes-1.15.1.tgz",
+      "integrity": "sha512-S6mL0yNB/Abt9Ei4tq8gDhcczc4S3+vQ4ra7vxnAf+YHC02srtqxKKZghx2Dq6p0e66THKwR6r8N6P95wEty7Q==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": "^20.19.0 || ^22.12.0 || >=24.0.0"
+      },
+      "peerDependencies": {
+        "@noble/hashes": "^1.8.0 || ^2.0.0"
+      },
+      "peerDependenciesMeta": {
+        "@noble/hashes": {
+          "optional": true
+        }
+      }
+    },
     "node_modules/@iconify/types": {
       "version": "2.0.0",
       "resolved": "https://registry.npmjs.org/@iconify/types/-/types-2.0.0.tgz",
       "integrity": "sha512-+wluvCrRhXrhyOmRDJ3q8mux9JkKy5SJ/v8ol2tu4FVjyYvtEzkc/3pK15ET6RKg4b4w4BmTk1+gsCUhf21Ykg==",
       "license": "MIT"
     },
     "node_modules/@iconify/utils": {
       "version": "3.1.7",
       "resolved": "https://registry.npmjs.org/@iconify/utils/-/utils-3.1.7.tgz",
       "integrity": "sha512-JZHlwdID+dy+lTgbYC8NEC4zeugqeYsc6jewvzb4c58kHauJn+X7rNwQjxz5p2qSjqaEeQoLkCIQ9v/H4PK0/w==",
@@ -1739,20 +1976,75 @@
     "node_modules/@tanstack/store": {
       "version": "0.11.1",
       "resolved": "https://registry.npmjs.org/@tanstack/store/-/store-0.11.1.tgz",
       "integrity": "sha512-mzTOBhypOuDJAy/D8n2MfUZ1HFkXnmSETviRyhqEC8LUE7/IZQExOTxMANj3KjTofYTkFNpBY67qaVrT41YccA==",
       "license": "MIT",
       "funding": {
         "type": "github",
         "url": "https://github.com/sponsors/tannerlinsley"
       }
     },
+    "node_modules/@testing-library/dom": {
+      "version": "10.4.2",
+      "resolved": "https://registry.npmjs.org/@testing-library/dom/-/dom-10.4.2.tgz",
+      "integrity": "sha512-yzr2S9HyAIdhz2/6qHgbs665Q7PKVcDF05vsOlHPxG1mo36gKVesdYVeDLnXgfjJ03CrKRk08knc6+E/9m8v2Q==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@babel/code-frame": "^7.10.4",
+        "@babel/runtime": "^7.12.5",
+        "@types/aria-query": "^5.0.1",
+        "aria-query": "5.3.0",
+        "dom-accessibility-api": "^0.5.9",
+        "lz-string": "^1.5.0",
+        "picocolors": "1.1.1",
+        "pretty-format": "^27.0.2"
+      },
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@testing-library/react": {
+      "version": "16.3.3",
+      "resolved": "https://registry.npmjs.org/@testing-library/react/-/react-16.3.3.tgz",
+      "integrity": "sha512-Uo193NgQbPMz6lrrhtRQQFcMC6Re/ELLFbbuVL30WDlZxlpZf9/lMHTAVxPRLw1q1iu9OJmR1c2BLiENRstdBg==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@babel/runtime": "^7.12.5"
+      },
+      "engines": {
+        "node": ">=18"
+      },
+      "peerDependencies": {
+        "@testing-library/dom": "^10.0.0",
+        "@types/react": "^18.0.0 || ^19.0.0",
+        "@types/react-dom": "^18.0.0 || ^19.0.0",
+        "react": "^18.0.0 || ^19.0.0",
+        "react-dom": "^18.0.0 || ^19.0.0"
+      },
+      "peerDependenciesMeta": {
+        "@types/react": {
+          "optional": true
+        },
+        "@types/react-dom": {
+          "optional": true
+        }
+      }
+    },
+    "node_modules/@types/aria-query": {
+      "version": "5.0.4",
+      "resolved": "https://registry.npmjs.org/@types/aria-query/-/aria-query-5.0.4.tgz",
+      "integrity": "sha512-rfT93uj5s0PRL7EzccGMs3brplhcrghnDoV26NqKhCAS1hVo+WdNsPvE/yb6ilfr5hi2MEk6d5EWJTKdxg8jVw==",
+      "dev": true,
+      "license": "MIT"
+    },
     "node_modules/@types/babel__core": {
       "version": "7.20.5",
       "resolved": "https://registry.npmjs.org/@types/babel__core/-/babel__core-7.20.5.tgz",
       "integrity": "sha512-qoQprZvz5wQFJwMDqeseRXWv3rqMvhgpbXFfVyWhbx9X47POIA6i/+dXefEmZKoAgOaTdaIgNSMqMIU61yRyzA==",
       "dev": true,
       "license": "MIT",
       "dependencies": {
         "@babel/parser": "^7.20.7",
         "@babel/types": "^7.20.7",
         "@types/babel__generator": "*",
@@ -2234,20 +2526,53 @@
     "node_modules/@vitest/spy": {
       "version": "5.0.1",
       "resolved": "https://registry.npmjs.org/@vitest/spy/-/spy-5.0.1.tgz",
       "integrity": "sha512-rbto/mF/SGERxEgYOek7Xm6B9b+y+mVoo+f4b2LymYO8zM1b7uB5nHuhVMTP2hxdzgxvGiZYGxGIaMvL5y180Q==",
       "dev": true,
       "license": "MIT",
       "funding": {
         "url": "https://opencollective.com/vitest"
       }
     },
+    "node_modules/ansi-regex": {
+      "version": "5.0.1",
+      "resolved": "https://registry.npmjs.org/ansi-regex/-/ansi-regex-5.0.1.tgz",
+      "integrity": "sha512-quJQXlTSUGL2LH9SUXo8VwsY4soanhgo6LNSm84E1LBcE8s3O0wpdiRzyR9z/ZZJMlMWv37qOOb9pdJlMUEKFQ==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=8"
+      }
+    },
+    "node_modules/ansi-styles": {
+      "version": "5.2.0",
+      "resolved": "https://registry.npmjs.org/ansi-styles/-/ansi-styles-5.2.0.tgz",
+      "integrity": "sha512-Cxwpt2SfTzTtXcfOlzGEee8O+c+MmUgGrNiBcXnuWxuFJHe6a5Hz7qwhwe5OgaSYI0IJvkLqWX1ASG+cJOkEiA==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=10"
+      },
+      "funding": {
+        "url": "https://github.com/chalk/ansi-styles?sponsor=1"
+      }
+    },
+    "node_modules/aria-query": {
+      "version": "5.3.0",
+      "resolved": "https://registry.npmjs.org/aria-query/-/aria-query-5.3.0.tgz",
+      "integrity": "sha512-b0P0sZPKtyu8HkeRAfCq0IfURZK+SuwMjY1UXGBU27wpAiTwQAIlq56IbIO+ytk/JjS1fMR14ee5WBBfKi5J6A==",
+      "dev": true,
+      "license": "Apache-2.0",
+      "dependencies": {
+        "dequal": "^2.0.3"
+      }
+    },
     "node_modules/assertion-error": {
       "version": "2.0.1",
       "resolved": "https://registry.npmjs.org/assertion-error/-/assertion-error-2.0.1.tgz",
       "integrity": "sha512-Izi8RQcffqCeNVgFigKli1ssklIbpHnCYc6AknXGYoB6grJqyeby7jv12JUQgmTAnIDnbck1uxksT4dzN3PWBA==",
       "dev": true,
       "license": "MIT",
       "engines": {
         "node": ">=12"
       }
     },
@@ -2267,20 +2592,30 @@
       "integrity": "sha512-hYrgxie335U08WqICoGqKRzV1HFXv6zdxwJE4ekCb80CM9a0SVVsN4QPwT67RraRo+9h8IATk6uxHJw7QSkdOg==",
       "dev": true,
       "license": "Apache-2.0",
       "bin": {
         "baseline-browser-mapping": "dist/cli.cjs"
       },
       "engines": {
         "node": ">=6.0.0"
       }
     },
+    "node_modules/bidi-js": {
+      "version": "1.1.0",
+      "resolved": "https://registry.npmjs.org/bidi-js/-/bidi-js-1.1.0.tgz",
+      "integrity": "sha512-fX1Onk0tdVPC7obPWB5EbJ1z7NVhLq4m2xZLq2YXBkxzMXIGRpNMU88n0EPgWseKl12J7zXs7qrDxPK4sRs2fg==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "require-from-string": "^2.0.2"
+      }
+    },
     "node_modules/browserslist": {
       "version": "4.29.0",
       "resolved": "https://registry.npmjs.org/browserslist/-/browserslist-4.29.0.tgz",
       "integrity": "sha512-3GSvyjvDI4Dur1Meg2BekJquu5uF+9R9a1+5M1Mde192eZoXbeXjzgOsgqPS2V8D5wrrip0gR5Hf/GhWQ9ZzaA==",
       "dev": true,
       "funding": [
         {
           "type": "opencollective",
           "url": "https://opencollective.com/browserslist"
         },
@@ -2437,20 +2772,34 @@
     },
     "node_modules/cose-base": {
       "version": "1.0.3",
       "resolved": "https://registry.npmjs.org/cose-base/-/cose-base-1.0.3.tgz",
       "integrity": "sha512-s9whTXInMSgAp/NVXVNuVxVKzGH2qck3aQlVHxDCdAEPgtMKwc4Wq6/QKhgdEdgbLSi9rBTAcPoRa6JpiG4ksg==",
       "license": "MIT",
       "dependencies": {
         "layout-base": "^1.0.0"
       }
     },
+    "node_modules/css-tree": {
+      "version": "3.2.1",
+      "resolved": "https://registry.npmjs.org/css-tree/-/css-tree-3.2.1.tgz",
+      "integrity": "sha512-X7sjQzceUhu1u7Y/ylrRZFU2FS6LRiFVp6rKLPg23y3x3c3DOKAwuXGDp+PAGjh6CSnCjYeAul8pcT8bAl+lSA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "mdn-data": "2.27.1",
+        "source-map-js": "^1.2.1"
+      },
+      "engines": {
+        "node": "^10 || ^12.20.0 || ^14.13.0 || >=15.0.0"
+      }
+    },
     "node_modules/csstype": {
       "version": "3.2.3",
       "resolved": "https://registry.npmjs.org/csstype/-/csstype-3.2.3.tgz",
       "integrity": "sha512-z1HGKcYy2xA8AGQfwrn0PAy+PB7X/GSj3UVJW9qKyn43xWa+gl5nXmU4qqLMRzWVLFC8KusUX8T/0kCiOYpAIQ==",
       "license": "MIT"
     },
     "node_modules/cytoscape": {
       "version": "3.34.3",
       "resolved": "https://registry.npmjs.org/cytoscape/-/cytoscape-3.34.3.tgz",
       "integrity": "sha512-yfYGhRcGAntq6YBD583j4n0Eg3jIxvWmZtz/5uz9UYkeIStSlMxuUja+ec5j3iBD8nv1rwaOAYMW09tBdkSeaQ==",
@@ -2942,20 +3291,49 @@
     "node_modules/dagre-d3-es": {
       "version": "7.0.14",
       "resolved": "https://registry.npmjs.org/dagre-d3-es/-/dagre-d3-es-7.0.14.tgz",
       "integrity": "sha512-P4rFMVq9ESWqmOgK+dlXvOtLwYg0i7u0HBGJER0LZDJT2VHIPAMZ/riPxqJceWMStH5+E61QxFra9kIS3AqdMg==",
       "license": "MIT",
       "dependencies": {
         "d3": "^7.9.0",
         "lodash-es": "^4.17.21"
       }
     },
+    "node_modules/data-urls": {
+      "version": "7.0.0",
+      "resolved": "https://registry.npmjs.org/data-urls/-/data-urls-7.0.0.tgz",
+      "integrity": "sha512-23XHcCF+coGYevirZceTVD7NdJOqVn+49IHyxgszm+JIiHLoB2TkmPtsYkNWT1pvRSGkc35L6NHs0yHkN2SumA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "whatwg-mimetype": "^5.0.0",
+        "whatwg-url": "^16.0.0"
+      },
+      "engines": {
+        "node": "^20.19.0 || ^22.12.0 || >=24.0.0"
+      }
+    },
+    "node_modules/data-urls/node_modules/whatwg-url": {
+      "version": "16.0.1",
+      "resolved": "https://registry.npmjs.org/whatwg-url/-/whatwg-url-16.0.1.tgz",
+      "integrity": "sha512-1to4zXBxmXHV3IiSSEInrreIlu02vUOvrhxJJH5vcxYTBDAx51cqZiKdyTxlecdKNSjj8EcxGBxNf6Vg+945gw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@exodus/bytes": "^1.11.0",
+        "tr46": "^6.0.0",
+        "webidl-conversions": "^8.0.1"
+      },
+      "engines": {
+        "node": "^20.19.0 || ^22.12.0 || >=24.0.0"
+      }
+    },
     "node_modules/dayjs": {
       "version": "1.11.23",
       "resolved": "https://registry.npmjs.org/dayjs/-/dayjs-1.11.23.tgz",
       "integrity": "sha512-QDTCU0M0MxR3hQfnlDJfwekQiaanm1ubOD231u73WBckQ/fsamwRLiE2GBz6D3a/xF1NgfiDLJjXBa1hYOYTtQ==",
       "license": "MIT"
     },
     "node_modules/debug": {
       "version": "4.4.3",
       "resolved": "https://registry.npmjs.org/debug/-/debug-4.4.3.tgz",
       "integrity": "sha512-RGwwWnwQvkVfavKVt22FGLw+xYSdzARwm0ru6DhTVA3umU5hZc28V3kO4stgYryrTlLpuvgI9GiijltAjNbcqA==",
@@ -2965,20 +3343,27 @@
       },
       "engines": {
         "node": ">=6.0"
       },
       "peerDependenciesMeta": {
         "supports-color": {
           "optional": true
         }
       }
     },
+    "node_modules/decimal.js": {
+      "version": "10.6.0",
+      "resolved": "https://registry.npmjs.org/decimal.js/-/decimal.js-10.6.0.tgz",
+      "integrity": "sha512-YpgQiITW3JXGntzdUmyUR1V812Hn8T1YVXhCu+wO3OpS4eU9l4YdD3qjyiKdV6mvV29zapkMeD390UVEf2lkUg==",
+      "dev": true,
+      "license": "MIT"
+    },
     "node_modules/decode-named-character-reference": {
       "version": "1.3.0",
       "resolved": "https://registry.npmjs.org/decode-named-character-reference/-/decode-named-character-reference-1.3.0.tgz",
       "integrity": "sha512-GtpQYB283KrPp6nRw50q3U9/VfOutZOe103qlN7BPP6Ad27xYnOIWv4lPzo8HCAL+mMZofJ9KEy30fq6MfaK6Q==",
       "license": "MIT",
       "dependencies": {
         "character-entities": "^2.0.0"
       },
       "funding": {
         "type": "github",
@@ -3019,20 +3404,27 @@
       "integrity": "sha512-RWmIqhcFf1lRYBvNmr7qTNuyCt/7/ns2jbpp1+PalgE/rDQcBT0fioSMUpJ93irlUhC5hrg4cYqe6U+0ImW0rA==",
       "license": "MIT",
       "dependencies": {
         "dequal": "^2.0.0"
       },
       "funding": {
         "type": "github",
         "url": "https://github.com/sponsors/wooorm"
       }
     },
+    "node_modules/dom-accessibility-api": {
+      "version": "0.5.16",
+      "resolved": "https://registry.npmjs.org/dom-accessibility-api/-/dom-accessibility-api-0.5.16.tgz",
+      "integrity": "sha512-X7BJ2yElsnOJ30pZF4uIIDfBEVgF4XEBxL9Bxhy6dnrm5hkzqmsWHGTiHqRiITNhMyFLyAiWndIJP7Z1NTteDg==",
+      "dev": true,
+      "license": "MIT"
+    },
     "node_modules/dompurify": {
       "version": "3.4.15",
       "resolved": "https://registry.npmjs.org/dompurify/-/dompurify-3.4.15.tgz",
       "integrity": "sha512-EUBjM+B+lkDE41iE82DDSCfkoPGfXx8IxFxPMjNzm/Uk4xDet77rTN9wqlxlVg71kK7XGuUMv6wUxJUwwv+Xyw==",
       "license": "(MPL-2.0 OR Apache-2.0)",
       "optionalDependencies": {
         "@types/trusted-types": "^2.0.7"
       }
     },
     "node_modules/electron-to-chromium": {
@@ -3055,20 +3447,33 @@
       "dev": true,
       "license": "MIT",
       "dependencies": {
         "graceful-fs": "^4.2.4",
         "tapable": "^2.3.3"
       },
       "engines": {
         "node": ">=10.13.0"
       }
     },
+    "node_modules/entities": {
+      "version": "8.1.0",
+      "resolved": "https://registry.npmjs.org/entities/-/entities-8.1.0.tgz",
+      "integrity": "sha512-kxL7msIffSuh9aaFAMD7rxAIuTRMAHMeBtgHW2yUdWw732ZNh4MehkF2gdjvtdmikkaIP9bFDDJOPlsvm7avrA==",
+      "dev": true,
+      "license": "BSD-2-Clause",
+      "engines": {
+        "node": ">=20.19.0"
+      },
+      "funding": {
+        "url": "https://github.com/fb55/entities?sponsor=1"
+      }
+    },
     "node_modules/es-module-lexer": {
       "version": "2.3.2",
       "resolved": "https://registry.npmjs.org/es-module-lexer/-/es-module-lexer-2.3.2.tgz",
       "integrity": "sha512-poHGpORABojJJucnV9KbOavETW8lBVnphkW77ER5/BQ5Fz7oXSoCNek7IH3vR5nRjdsEz926ibFYX8KtLQmdyw==",
       "dev": true,
       "license": "MIT"
     },
     "node_modules/es-toolkit": {
       "version": "1.52.0",
       "resolved": "https://registry.npmjs.org/es-toolkit/-/es-toolkit-1.52.0.tgz",
@@ -3361,20 +3766,33 @@
       "integrity": "sha512-88JUN06ipLwsnv+dVn+OIYOvAuvBMy/Qoi6O7mQHxdPXpjy+Cd6xRkWwux7DKO+4sYILtLBRIKgsdpS2gQc7qw==",
       "license": "MIT",
       "dependencies": {
         "@types/hast": "^3.0.0"
       },
       "funding": {
         "type": "opencollective",
         "url": "https://opencollective.com/unified"
       }
     },
+    "node_modules/html-encoding-sniffer": {
+      "version": "6.0.0",
+      "resolved": "https://registry.npmjs.org/html-encoding-sniffer/-/html-encoding-sniffer-6.0.0.tgz",
+      "integrity": "sha512-CV9TW3Y3f8/wT0BRFc1/KAVQ3TUHiXmaAb6VW9vtiMFf7SLoMd1PdAc4W3KFOFETBJUb90KatHqlsZMWV+R9Gg==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@exodus/bytes": "^1.6.0"
+      },
+      "engines": {
+        "node": "^20.19.0 || ^22.12.0 || >=24.0.0"
+      }
+    },
     "node_modules/html-url-attributes": {
       "version": "3.0.1",
       "resolved": "https://registry.npmjs.org/html-url-attributes/-/html-url-attributes-3.0.1.tgz",
       "integrity": "sha512-ol6UPyBWqsrO6EJySPz2O7ZSr856WDrEzM5zMqp+FJJLGMW35cLYmmZnl0vztAZxRUoNZJFTCohfjuIJ8I4QBQ==",
       "license": "MIT",
       "funding": {
         "type": "opencollective",
         "url": "https://opencollective.com/unified"
       }
     },
@@ -3464,20 +3882,27 @@
       "resolved": "https://registry.npmjs.org/is-plain-obj/-/is-plain-obj-4.1.0.tgz",
       "integrity": "sha512-+Pgi+vMuUNkJyExiMBt5IlFoMyKnr5zhJ4Uspz58WOhBF5QoIZkFyNHIbBAtHwzVAgk5RtndVNsDRN61/mmDqg==",
       "license": "MIT",
       "engines": {
         "node": ">=12"
       },
       "funding": {
         "url": "https://github.com/sponsors/sindresorhus"
       }
     },
+    "node_modules/is-potential-custom-element-name": {
+      "version": "1.0.1",
+      "resolved": "https://registry.npmjs.org/is-potential-custom-element-name/-/is-potential-custom-element-name-1.0.1.tgz",
+      "integrity": "sha512-bCYeRA2rVibKZd+s2625gGnGF/t7DSqDs4dP7CrLA1m7jKWz6pps0LpYLJN8Q64HtmPKJ1hrN3nzPNKFEKOUiQ==",
+      "dev": true,
+      "license": "MIT"
+    },
     "node_modules/isbot": {
       "version": "5.2.2",
       "resolved": "https://registry.npmjs.org/isbot/-/isbot-5.2.2.tgz",
       "integrity": "sha512-iQcBXcd+Rv/pkubRyGh2utW2j1oPG5hZY6TUhVPpqK4G+o3IbxpJNx04hgksjc/N7GK5pEorUxDeg31cFgEk/w==",
       "license": "Unlicense",
       "engines": {
         "node": ">=18"
       }
     },
     "node_modules/jiti": {
@@ -3490,20 +3915,70 @@
         "jiti": "lib/jiti-cli.mjs"
       }
     },
     "node_modules/js-tokens": {
       "version": "4.0.0",
       "resolved": "https://registry.npmjs.org/js-tokens/-/js-tokens-4.0.0.tgz",
       "integrity": "sha512-RdJUflcE3cUzKiMqQgsCu06FPu9UdIJO0beYbPhHN4k6apgJtifcoCtT9bcxOpYBtpD2kCM6Sbzg4CausW/PKQ==",
       "dev": true,
       "license": "MIT"
     },
+    "node_modules/jsdom": {
+      "version": "30.1.0",
+      "resolved": "https://registry.npmjs.org/jsdom/-/jsdom-30.1.0.tgz",
+      "integrity": "sha512-h/Q+H+KmlGf49ITw3TrykSVbg04S75S2O7ZtJc9gMGOWBrQRZmx5AJIKLtJ01e6Kszl+/tpFthCVqxNifxhoqg==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@asamuzakjp/css-color": "^7.0.0",
+        "@asamuzakjp/dom-selector": "^9.1.2",
+        "@bramus/specificity": "^2.4.2",
+        "@csstools/css-syntax-patches-for-csstree": "^1.1.13",
+        "@exodus/bytes": "^1.15.1",
+        "css-tree": "^3.2.1",
+        "data-urls": "^7.0.0",
+        "decimal.js": "^10.6.0",
+        "html-encoding-sniffer": "^6.0.0",
+        "is-potential-custom-element-name": "^1.0.1",
+        "lru-cache": "^11.5.2",
+        "parse5": "^8.0.1",
+        "saxes": "^6.0.0",
+        "tough-cookie": "^6.0.2",
+        "undici": "^8.10.2",
+        "w3c-xmlserializer": "^5.0.0",
+        "webidl-conversions": "^8.0.1",
+        "whatwg-mimetype": "^5.0.0",
+        "whatwg-url": "^17.1.1",
+        "xml-name-validator": "^5.0.0"
+      },
+      "engines": {
+        "node": "^22.22.2 || ^24.15.0 || >=26.0.0"
+      },
+      "peerDependencies": {
+        "canvas": "^3.2.3"
+      },
+      "peerDependenciesMeta": {
+        "canvas": {
+          "optional": true
+        }
+      }
+    },
+    "node_modules/jsdom/node_modules/lru-cache": {
+      "version": "11.5.3",
+      "resolved": "https://registry.npmjs.org/lru-cache/-/lru-cache-11.5.3.tgz",
+      "integrity": "sha512-U4N8FgzmWxc8k1VH8Kr6lQg18U7Fjvby6wXHVRX/ZZ7IwWbRMgrRbP0Wrb5q5NVinryp4SQampHKdvtecItxUg==",
+      "dev": true,
+      "license": "BlueOak-1.0.0",
+      "engines": {
+        "node": "20 || >=22"
+      }
+    },
     "node_modules/jsesc": {
       "version": "3.1.0",
       "resolved": "https://registry.npmjs.org/jsesc/-/jsesc-3.1.0.tgz",
       "integrity": "sha512-/sM3dO2FOzXjKQhJuo0Q173wf2KOo8t4I8vHy6lF9poUp7bKT0/NHE8fPX23PwfhnykfqnC2xRxOnVw5XuGIaA==",
       "dev": true,
       "license": "MIT",
       "bin": {
         "jsesc": "bin/jsesc"
       },
       "engines": {
@@ -3851,20 +4326,30 @@
     "node_modules/lru-cache": {
       "version": "5.1.1",
       "resolved": "https://registry.npmjs.org/lru-cache/-/lru-cache-5.1.1.tgz",
       "integrity": "sha512-KpNARQA3Iwv+jTA0utUVVbrh+Jlrr1Fv0e56GGzAFOXN7dk/FviaDW8LHmK52DlcH4WP2n6gI8vN1aesBFgo9w==",
       "dev": true,
       "license": "ISC",
       "dependencies": {
         "yallist": "^3.0.2"
       }
     },
+    "node_modules/lz-string": {
+      "version": "1.5.0",
+      "resolved": "https://registry.npmjs.org/lz-string/-/lz-string-1.5.0.tgz",
+      "integrity": "sha512-h5bgJWpxJNswbU7qCrV0tIKQCaS3blPDrqKWx+QxzuzL1zGUzij9XCWLrSLsJPu5t+eWA/ycetzYAO5IOMcWAQ==",
+      "dev": true,
+      "license": "MIT",
+      "bin": {
+        "lz-string": "bin/bin.js"
+      }
+    },
     "node_modules/magic-string": {
       "version": "0.30.21",
       "resolved": "https://registry.npmjs.org/magic-string/-/magic-string-0.30.21.tgz",
       "integrity": "sha512-vd2F4YUyEXKGcLHoq+TEyCjxueSeHnFxyyjNp80yg0XV4vUhnDer/lvvlqM/arB5bXQN5K2/3oinyCRyx8T2CQ==",
       "dev": true,
       "license": "MIT",
       "dependencies": {
         "@jridgewell/sourcemap-codec": "^1.5.5"
       }
     },
@@ -4153,20 +4638,27 @@
       "integrity": "sha512-0H44vDimn51F0YwvxSJSm0eCDOJTRlmN0R1yBh4HLj9wiV1Dn0QoXGbvFAWj2hSItVTlCmBF1hqKlIyUBVFLPg==",
       "license": "MIT",
       "dependencies": {
         "@types/mdast": "^4.0.0"
       },
       "funding": {
         "type": "opencollective",
         "url": "https://opencollective.com/unified"
       }
     },
+    "node_modules/mdn-data": {
+      "version": "2.27.1",
+      "resolved": "https://registry.npmjs.org/mdn-data/-/mdn-data-2.27.1.tgz",
+      "integrity": "sha512-9Yubnt3e8A0OKwxYSXyhLymGW4sCufcLG6VdiDdUGVkPhpqLxlvP5vl1983gQjJl3tqbrM731mjaZaP68AgosQ==",
+      "dev": true,
+      "license": "CC0-1.0"
+    },
     "node_modules/mermaid": {
       "version": "12.0.0",
       "resolved": "https://registry.npmjs.org/mermaid/-/mermaid-12.0.0.tgz",
       "integrity": "sha512-/wQXC9iBxoGV8p3erbvaXs9h77VyLDBH6GdayVjj3hEcSQhFU4N1WUhUppotCEqlIxI2pRMwjwBSwTB1MfZBgQ==",
       "license": "MIT",
       "dependencies": {
         "@braintree/sanitize-url": "^7.1.2",
         "@iconify/utils": "^3.0.2",
         "@mermaid-js/parser": "^2.0.0",
         "@types/d3": "^7.4.3",
@@ -4854,20 +5346,33 @@
         "type": "github",
         "url": "https://github.com/sponsors/wooorm"
       }
     },
     "node_modules/parse-entities/node_modules/@types/unist": {
       "version": "2.0.11",
       "resolved": "https://registry.npmjs.org/@types/unist/-/unist-2.0.11.tgz",
       "integrity": "sha512-CmBKiL6NNo/OqgmMn95Fk9Whlp2mtvIv+KNpQKN2F4SjvrEesubTRWGYSg+BnWZOnlCaSTU1sMpsBOzgbYhnsA==",
       "license": "MIT"
     },
+    "node_modules/parse5": {
+      "version": "8.0.1",
+      "resolved": "https://registry.npmjs.org/parse5/-/parse5-8.0.1.tgz",
+      "integrity": "sha512-z1e/HMG90obSGeidlli3hj7cbocou0/wa5HacvI3ASx34PecNjNQeaHNo5WIZpWofN9kgkqV1q5YvXe3F0FoPw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "entities": "^8.0.0"
+      },
+      "funding": {
+        "url": "https://github.com/inikulin/parse5?sponsor=1"
+      }
+    },
     "node_modules/path-data-parser": {
       "version": "0.1.0",
       "resolved": "https://registry.npmjs.org/path-data-parser/-/path-data-parser-0.1.0.tgz",
       "integrity": "sha512-NOnmBpt5Y2RWbuv0LMzsayp3lVylAHLPUTut412ZA3l+C4uw4ZVkQbjShYCQ8TCpUMdPapr4YjUqLYD6v68j+w==",
       "license": "MIT"
     },
     "node_modules/picocolors": {
       "version": "1.1.1",
       "resolved": "https://registry.npmjs.org/picocolors/-/picocolors-1.1.1.tgz",
       "integrity": "sha512-xceH2snhtb5M9liqDsmEw56le376mTZkEX/jEb/RxNFyegNul7eNslCXP9FDj/Lcu0X8KEyMceP2ntpaHrDEVA==",
@@ -4925,30 +5430,55 @@
       "license": "MIT",
       "dependencies": {
         "nanoid": "^3.3.18",
         "picocolors": "^1.1.1",
         "source-map-js": "^1.2.1"
       },
       "engines": {
         "node": "^10 || ^12 || >=14"
       }
     },
+    "node_modules/pretty-format": {
+      "version": "27.5.1",
+      "resolved": "https://registry.npmjs.org/pretty-format/-/pretty-format-27.5.1.tgz",
+      "integrity": "sha512-Qb1gy5OrP5+zDf2Bvnzdl3jsTf1qXVMazbvCoKhtKqVs4/YK4ozX4gKQJJVyNe+cajNPn0KoC0MC3FUmaHWEmQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "ansi-regex": "^5.0.1",
+        "ansi-styles": "^5.0.0",
+        "react-is": "^17.0.1"
+      },
+      "engines": {
+        "node": "^10.13.0 || ^12.13.0 || ^14.15.0 || >=15.0.0"
+      }
+    },
     "node_modules/property-information": {
       "version": "7.2.0",
       "resolved": "https://registry.npmjs.org/property-information/-/property-information-7.2.0.tgz",
       "integrity": "sha512-IAtzIB6sUiWaJYrX9smp3V46pBGbBeLFRGdh25kg1334VcBlD8HzhPeNIWQH9zhGmo2itIe25EHt9dQP7G5hmg==",
       "license": "MIT",
       "funding": {
         "type": "github",
         "url": "https://github.com/sponsors/wooorm"
       }
     },
+    "node_modules/punycode": {
+      "version": "2.3.1",
+      "resolved": "https://registry.npmjs.org/punycode/-/punycode-2.3.1.tgz",
+      "integrity": "sha512-vYt7UD1U9Wg6138shLtLOvdAu+8DsC/ilFtEVHcH+wydcSpNE20AfSOduf6MkRFahL5FY7X1oU7nKVZFtfq8Fg==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=6"
+      }
+    },
     "node_modules/react": {
       "version": "19.3.0",
       "resolved": "https://registry.npmjs.org/react/-/react-19.3.0.tgz",
       "integrity": "sha512-E8LUcbtBWt20bbl2YoHfx4ZDBdxVTfOKtCZn9cDSJ4l6/nuoApcpIBcj47t2wZoVX8g2ZHuMHbiShgCR1T5Sog==",
       "license": "MIT",
       "engines": {
         "node": ">=0.10.0"
       }
     },
     "node_modules/react-dom": {
@@ -4956,20 +5486,27 @@
       "resolved": "https://registry.npmjs.org/react-dom/-/react-dom-19.3.0.tgz",
       "integrity": "sha512-JDk8dgif51OjFoDE70+OT9ICyYr+69HlmihNwp1+Nsfbna3t5sIiCa9ZJktDmQ4/1b/rn26hIAR2uYXDMr5r0Q==",
       "license": "MIT",
       "dependencies": {
         "scheduler": "^0.28.0"
       },
       "peerDependencies": {
         "react": "^19.3.0"
       }
     },
+    "node_modules/react-is": {
+      "version": "17.0.2",
+      "resolved": "https://registry.npmjs.org/react-is/-/react-is-17.0.2.tgz",
+      "integrity": "sha512-w2GsyukL62IJnlaff/nRegPQR94C/XXamvMWmSHRJ4y7Ts/4ocGRmTHvOs8PSE6pB3dWOrD/nueuU5sduBsQ4w==",
+      "dev": true,
+      "license": "MIT"
+    },
     "node_modules/react-markdown": {
       "version": "10.1.0",
       "resolved": "https://registry.npmjs.org/react-markdown/-/react-markdown-10.1.0.tgz",
       "integrity": "sha512-qKxVopLT/TyA6BX3Ue5NwabOsAzm0Q7kAPwq6L+wWDwisYs7R8vZ0nRXqq6rkueboxpkjvLGU9fWifiX/ZZFxQ==",
       "license": "MIT",
       "dependencies": {
         "@types/hast": "^3.0.0",
         "@types/mdast": "^4.0.0",
         "devlop": "^1.0.0",
         "hast-util-to-jsx-runtime": "^2.0.0",
@@ -5073,20 +5610,30 @@
       "dependencies": {
         "@types/mdast": "^4.0.0",
         "mdast-util-to-markdown": "^2.0.0",
         "unified": "^11.0.0"
       },
       "funding": {
         "type": "opencollective",
         "url": "https://opencollective.com/unified"
       }
     },
+    "node_modules/require-from-string": {
+      "version": "2.0.2",
+      "resolved": "https://registry.npmjs.org/require-from-string/-/require-from-string-2.0.2.tgz",
+      "integrity": "sha512-Xf0nWe6RseziFMu+Ap9biiUbmplq6S9/p+7w7YXP/JBHhrUDDUhwa+vANyubuqfZWTveU//DYVGsDG7RKL/vEw==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=0.10.0"
+      }
+    },
     "node_modules/robust-predicates": {
       "version": "3.0.3",
       "resolved": "https://registry.npmjs.org/robust-predicates/-/robust-predicates-3.0.3.tgz",
       "integrity": "sha512-NS3levdsRIUOmiJ8FZWCP7LG3QpJyrs/TE0Zpf1yvZu8cAJJ6QMW92H1c7kWpdIHo8RvmLxN/o2JXTKHp74lUA==",
       "license": "Unlicense"
     },
     "node_modules/rollup": {
       "version": "4.63.3",
       "resolved": "https://registry.npmjs.org/rollup/-/rollup-4.63.3.tgz",
       "integrity": "sha512-1i2XreiAoMMXuPGD6Msj2xWrMMkHojNRKivInxGQcg7/1KuPuYlfUutLyh4drnOxUTHX9cHI4wFoat8D/NKaBw==",
@@ -5149,20 +5696,33 @@
       "resolved": "https://registry.npmjs.org/rw/-/rw-1.3.3.tgz",
       "integrity": "sha512-PdhdWy89SiZogBLaw42zdeqtRJ//zFd2PgQavcICDUgJT5oW10QCRKbJ6bg4r0/UY2M6BWd5tkxuGFRvCkgfHQ==",
       "license": "BSD-3-Clause"
     },
     "node_modules/safer-buffer": {
       "version": "2.1.2",
       "resolved": "https://registry.npmjs.org/safer-buffer/-/safer-buffer-2.1.2.tgz",
       "integrity": "sha512-YZo3K82SD7Riyi0E1EQPojLz7kpepnSQI9IyPbHHg1XXXevb5dJI7tpyN2ADxGcQbHG7vcyRHk0cbwqcQriUtg==",
       "license": "MIT"
     },
+    "node_modules/saxes": {
+      "version": "6.0.0",
+      "resolved": "https://registry.npmjs.org/saxes/-/saxes-6.0.0.tgz",
+      "integrity": "sha512-xAg7SOnEhrm5zI3puOOKyy1OMcMlIJZYNJY7xLBwSze0UjhPLnWfj2GF2EpT0jmzaJKIWKHLsaSSajf35bcYnA==",
+      "dev": true,
+      "license": "ISC",
+      "dependencies": {
+        "xmlchars": "^2.2.0"
+      },
+      "engines": {
+        "node": ">=v12.22.7"
+      }
+    },
     "node_modules/scheduler": {
       "version": "0.28.0",
       "resolved": "https://registry.npmjs.org/scheduler/-/scheduler-0.28.0.tgz",
       "integrity": "sha512-juorfCmIkIw8tT+p5BXSm6PJjQF/ycEYmKyzURCIt/RaZIhL+PulbQ9Yu2z1HdOJDdqDTlxA1+xKBmHXJsczAw==",
       "license": "MIT"
     },
     "node_modules/semver": {
       "version": "6.3.1",
       "resolved": "https://registry.npmjs.org/semver/-/semver-6.3.1.tgz",
       "integrity": "sha512-BR7VvDCVHO+q2xBEWskxS6DJE1qRnb7DxzUrogb71CWoSficBxYsiAGd+Kl0mmq/MprG9yArRkyrQxTO6XjMzA==",
@@ -5332,20 +5892,66 @@
         "fdir": "^6.5.0",
         "picomatch": "^4.0.4"
       },
       "engines": {
         "node": ">=12.0.0"
       },
       "funding": {
         "url": "https://github.com/sponsors/SuperchupuDev"
       }
     },
+    "node_modules/tldts": {
+      "version": "7.4.13",
+      "resolved": "https://registry.npmjs.org/tldts/-/tldts-7.4.13.tgz",
+      "integrity": "sha512-iHtaIWWIbMDkCeJdTBzZFGgbluE5J+oHlb2g7+oAz1S1gpuVpabRZdQyd471Vl8UUkcz2vXSL8xZH2kyCe8tfA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "tldts-core": "^7.4.13"
+      },
+      "bin": {
+        "tldts": "bin/cli.js"
+      }
+    },
+    "node_modules/tldts-core": {
+      "version": "7.4.13",
+      "resolved": "https://registry.npmjs.org/tldts-core/-/tldts-core-7.4.13.tgz",
+      "integrity": "sha512-mbYsrih5FRtGxs3Usvl/PqwJsNpp+jsmrdFviiK02teHDG0/HebBG/pqCylje3kzgXYzuLoHJF/0mz9W53t8Xg==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/tough-cookie": {
+      "version": "6.0.2",
+      "resolved": "https://registry.npmjs.org/tough-cookie/-/tough-cookie-6.0.2.tgz",
+      "integrity": "sha512-exgYmnmL/sJpR3upZfXG5PoatXQii55xAiXGXzY+sROLZ/Y+SLcp9PgJNI9Vz37HpQ74WvDcLT8eqm+kV3FzrA==",
+      "dev": true,
+      "license": "BSD-3-Clause",
+      "dependencies": {
+        "tldts": "^7.0.5"
+      },
+      "engines": {
+        "node": ">=16"
+      }
+    },
+    "node_modules/tr46": {
+      "version": "6.0.0",
+      "resolved": "https://registry.npmjs.org/tr46/-/tr46-6.0.0.tgz",
+      "integrity": "sha512-bLVMLPtstlZ4iMQHpFHTR7GAGj2jxi8Dg0s2h2MafAE4uSWF98FC/3MomU51iQAMf8/qDUbKWf5GxuvvVcXEhw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "punycode": "^2.3.1"
+      },
+      "engines": {
+        "node": ">=20"
+      }
+    },
     "node_modules/trim-lines": {
       "version": "3.0.1",
       "resolved": "https://registry.npmjs.org/trim-lines/-/trim-lines-3.0.1.tgz",
       "integrity": "sha512-kRj8B+YHZCc9kQYdWfJB2/oUl9rA99qbowYYBtr4ui4mZyAQ2JpvVBd/6U2YloATfqBhBTSMhTpgBHtU0Mf3Rg==",
       "license": "MIT",
       "funding": {
         "type": "github",
         "url": "https://github.com/sponsors/wooorm"
       }
     },
@@ -5381,20 +5987,30 @@
       "dev": true,
       "license": "Apache-2.0",
       "bin": {
         "tsc": "bin/tsc",
         "tsserver": "bin/tsserver"
       },
       "engines": {
         "node": ">=14.17"
       }
     },
+    "node_modules/undici": {
+      "version": "8.10.2",
+      "resolved": "https://registry.npmjs.org/undici/-/undici-8.10.2.tgz",
+      "integrity": "sha512-/y4/bH9YNU5hi9NIrpOuvGXFcxrj3CMrV+/AYpowAYTpHn8gX/XPFjNy766FPoYY0miQhdW977JFWKGNhBdwyQ==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=22.19.0"
+      }
+    },
     "node_modules/unified": {
       "version": "11.0.5",
       "resolved": "https://registry.npmjs.org/unified/-/unified-11.0.5.tgz",
       "integrity": "sha512-xKvGhPWw3k84Qjh8bI3ZeJjqnyadK+GEFtazSfZv/rKeTkTjOJho6mFqh2SM96iIcZokxiOpg78GazTSg8+KHA==",
       "license": "MIT",
       "dependencies": {
         "@types/unist": "^3.0.0",
         "bail": "^2.0.0",
         "devlop": "^1.0.0",
         "extend": "^3.0.0",
@@ -5727,37 +6343,102 @@
     "node_modules/vitest/node_modules/tinyexec": {
       "version": "1.3.0",
       "resolved": "https://registry.npmjs.org/tinyexec/-/tinyexec-1.3.0.tgz",
       "integrity": "sha512-QKAl9m8gWWGHV8jZcPeym6j+XULi6tOf1mT83WYJ4Lk2ytW/uwAWkrP0uFsdoYMdueVJ0qs26wZ+23xeB4ibNQ==",
       "dev": true,
       "license": "MIT",
       "engines": {
         "node": ">=18"
       }
     },
+    "node_modules/w3c-xmlserializer": {
+      "version": "5.0.0",
+      "resolved": "https://registry.npmjs.org/w3c-xmlserializer/-/w3c-xmlserializer-5.0.0.tgz",
+      "integrity": "sha512-o8qghlI8NZHU1lLPrpi2+Uq7abh4GGPpYANlalzWxyWteJOCsr/P+oPBA49TOLu5FTZO4d3F9MnWJfiMo4BkmA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "xml-name-validator": "^5.0.0"
+      },
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/webidl-conversions": {
+      "version": "8.0.1",
+      "resolved": "https://registry.npmjs.org/webidl-conversions/-/webidl-conversions-8.0.1.tgz",
+      "integrity": "sha512-BMhLD/Sw+GbJC21C/UgyaZX41nPt8bUTg+jWyDeg7e7YN4xOM05YPSIXceACnXVtqyEw/LMClUQMtMZ+PGGpqQ==",
+      "dev": true,
+      "license": "BSD-2-Clause",
+      "engines": {
+        "node": ">=20"
+      }
+    },
+    "node_modules/whatwg-mimetype": {
+      "version": "5.0.0",
+      "resolved": "https://registry.npmjs.org/whatwg-mimetype/-/whatwg-mimetype-5.0.0.tgz",
+      "integrity": "sha512-sXcNcHOC51uPGF0P/D4NVtrkjSU2fNsm9iog4ZvZJsL3rjoDAzXZhkm2MWt1y+PUdggKAYVoMAIYcs78wJ51Cw==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=20"
+      }
+    },
+    "node_modules/whatwg-url": {
+      "version": "17.1.2",
+      "resolved": "https://registry.npmjs.org/whatwg-url/-/whatwg-url-17.1.2.tgz",
+      "integrity": "sha512-TEZA+Zqxin7Jjsm2cjRohCmen5awh+hT6Zi3VZdqZlNRk7zvOI/9WpBFg/DWlA56bWnzwm6DuB8NS0EsxQH9uQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@exodus/bytes": "^1.15.1",
+        "tr46": "^6.0.0",
+        "webidl-conversions": "^8.0.1"
+      },
+      "engines": {
+        "node": "^22.14.0 || >=24.0.0"
+      }
+    },
     "node_modules/why-is-node-running": {
       "version": "2.3.0",
       "resolved": "https://registry.npmjs.org/why-is-node-running/-/why-is-node-running-2.3.0.tgz",
       "integrity": "sha512-hUrmaWBdVDcxvYqnyh09zunKzROWjbZTiNy8dBEjkS7ehEDQibXJ7XvlmtbwuTclUiIyN+CyXQD4Vmko8fNm8w==",
       "dev": true,
       "license": "MIT",
       "dependencies": {
         "siginfo": "^2.0.0",
         "stackback": "0.0.2"
       },
       "bin": {
         "why-is-node-running": "cli.js"
       },
       "engines": {
         "node": ">=8"
       }
     },
+    "node_modules/xml-name-validator": {
+      "version": "5.0.0",
+      "resolved": "https://registry.npmjs.org/xml-name-validator/-/xml-name-validator-5.0.0.tgz",
+      "integrity": "sha512-EvGK8EJ3DhaHfbRlETOWAS5pO9MZITeauHKJyb8wyajUfQUenkIg2MvLDTZ4T/TgIcm3HU0TFBgWWboAZ30UHg==",
+      "dev": true,
+      "license": "Apache-2.0",
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/xmlchars": {
+      "version": "2.2.0",
+      "resolved": "https://registry.npmjs.org/xmlchars/-/xmlchars-2.2.0.tgz",
+      "integrity": "sha512-JZnDKK8B0RCDw84FNdDAIpZK+JuJw+s7Lz8nksI7SIuU3UXJJslUthsi+uWBUYOwPFwW7W7PRLRfUKpxjtjFCw==",
+      "dev": true,
+      "license": "MIT"
+    },
     "node_modules/yallist": {
       "version": "3.1.1",
       "resolved": "https://registry.npmjs.org/yallist/-/yallist-3.1.1.tgz",
       "integrity": "sha512-a4UGQaWPH59mOXUYnAG2ewncQS4i4F43Tv3JoAM+s2VDAmS9NsK8GpDMLrCHPksFT7h3K6TOoUNn2pb7RoXx4g==",
       "dev": true,
       "license": "ISC"
     },
     "node_modules/zwitch": {
       "version": "2.0.4",
       "resolved": "https://registry.npmjs.org/zwitch/-/zwitch-2.0.4.tgz",
diff --git a/skillgrid-ui/package.json b/skillgrid-ui/package.json
index 34a1511..5a58bbc 100644
--- a/skillgrid-ui/package.json
+++ b/skillgrid-ui/package.json
@@ -25,19 +25,22 @@
     "mermaid": "^12.0.0",
     "react": "^19.1.0",
     "react-dom": "^19.1.0",
     "react-markdown": "^10.1.0",
     "rehype-sanitize": "^6.0.0",
     "remark-gfm": "^4.0.1",
     "sigma": "^3.0.3"
   },
   "devDependencies": {
     "@tailwindcss/vite": "^4.1.11",
+    "@testing-library/dom": "^10.4.2",
+    "@testing-library/react": "^16.3.3",
     "@types/react": "^19.1.8",
     "@types/react-dom": "^19.1.6",
     "@vitejs/plugin-react": "^4.6.0",
+    "jsdom": "^30.1.0",
     "tailwindcss": "^4.1.11",
     "typescript": "~5.8.3",
     "vite": "^6.3.5",
     "vitest": "^5.0.1"
   }
 }
diff --git a/skillgrid-ui/src/app.tsx b/skillgrid-ui/src/app.tsx
index acffb39..27583c1 100644
--- a/skillgrid-ui/src/app.tsx
+++ b/skillgrid-ui/src/app.tsx
@@ -84,20 +84,32 @@ const plansRoute = createRoute({
   path: '/plans',
   component: lazyPage(() => import('./features/plans/PlansPage').then((m) => ({ default: m.PlansPage }))),
 })
 
 const activityRoute = createRoute({
   getParentRoute: () => rootRoute,
   path: '/activity',
   component: lazyPage(() => import('./features/activity/ActivityPage').then((m) => ({ default: m.ActivityPage }))),
 })
 
+const handoffRoute = createRoute({
+  getParentRoute: () => rootRoute,
+  path: '/handoff',
+  component: lazyPage(() => import('./features/handoff/HandoffHubPage').then((m) => ({ default: m.HandoffHubPage }))),
+})
+
+const decisionsRoute = createRoute({
+  getParentRoute: () => rootRoute,
+  path: '/decisions',
+  component: lazyPage(() => import('./features/decisions/DecisionsPage').then((m) => ({ default: m.DecisionsPage }))),
+})
+
 const gitRoute = createRoute({
   getParentRoute: () => rootRoute,
   path: '/git',
   component: lazyPage(() => import('./features/git/GitPage').then((m) => ({ default: m.GitPage }))),
 })
 
 const prototypesRoute = createRoute({
   getParentRoute: () => rootRoute,
   path: '/prototypes',
   component: lazyPage(() => import('./features/prototypes/PrototypesPage').then((m) => ({ default: m.PrototypesPage }))),
@@ -119,20 +131,22 @@ const routeTree = rootRoute.addChildren([
   indexRoute,
   trackerRoute,
   graphRoute,
   filesRoute,
   memoriesRoute,
   sessionsRoute,
   searchRoute,
   docsRoute,
   plansRoute,
   activityRoute,
+  handoffRoute,
+  decisionsRoute,
   gitRoute,
   prototypesRoute,
   settingsRoute,
   notFoundRoute,
 ])
 
 const router = createRouter({ routeTree })
 
 export function App() {
   return <RouterProvider router={router} />
diff --git a/skillgrid-ui/src/components/layout/AppLayout.tsx b/skillgrid-ui/src/components/layout/AppLayout.tsx
index ede8e06..a5e5392 100644
--- a/skillgrid-ui/src/components/layout/AppLayout.tsx
+++ b/skillgrid-ui/src/components/layout/AppLayout.tsx
@@ -7,21 +7,23 @@ const MNEMONIC_ITEMS: Array<{ to: string; label: string }> = [
   { to: '/mnemonic/graph', label: 'Graph' },
   { to: '/mnemonic/files', label: 'Files' },
   { to: '/mnemonic/memories', label: 'Memories' },
   { to: '/mnemonic/sessions', label: 'Sessions' },
   { to: '/mnemonic/search', label: 'Search' },
 ]
 
 const STANDALONE_ITEMS: Array<{ to: string; label: string }> = [
   { to: '/docs', label: 'Docs' },
   { to: '/plans', label: 'Plans' },
+  { to: '/handoff', label: 'Handoff' },
   { to: '/activity', label: 'Activity' },
+  { to: '/decisions', label: 'Decisions' },
   { to: '/git', label: 'Git' },
   { to: '/prototypes', label: 'Prototypes' },
   { to: '/settings', label: 'Settings' },
 ]
 
 export function AppLayout() {
   // Phase 7.4 responsive: the sidebar is a slide-over on mobile (hidden by
   // default, opened via the hamburger) and a fixed column on lg+ screens.
   const [navOpen, setNavOpen] = useState(false)
   const hamburgerRef = useRef<HTMLButtonElement>(null)
diff --git a/skillgrid-ui/src/features/decisions/DecisionCard.test.tsx b/skillgrid-ui/src/features/decisions/DecisionCard.test.tsx
new file mode 100644
index 0000000..3e63c8c
--- /dev/null
+++ b/skillgrid-ui/src/features/decisions/DecisionCard.test.tsx
@@ -0,0 +1,65 @@
+import { render, screen, fireEvent } from '@testing-library/react'
+import { describe, it, expect, vi } from 'vitest'
+import { DecisionCard, type Decision } from './DecisionCard'
+
+const pending: Decision = {
+  id: 1,
+  topicKey: 'interview/demo/layout-1',
+  title: 'Layout',
+  createdAt: '2026-09-19T10:00:00Z',
+  updatedAt: '2026-09-19T10:00:00Z',
+  visibility: 'team',
+  content: {
+    question: 'Which layout?',
+    options: [
+      { id: 'a', label: 'Single column' },
+      { id: 'b', label: 'Two column' },
+    ],
+    recommended: 'a',
+    state: 'pending',
+  },
+}
+
+const answered: Decision = {
+  ...pending,
+  id: 2,
+  content: {
+    ...pending.content,
+    state: 'answered',
+    answeredOption: 'b',
+    answerNote: 'picked two column for density',
+  },
+}
+
+describe('DecisionCard', () => {
+  it('renders options with the recommended one highlighted', () => {
+    render(<DecisionCard decision={pending} onAnswer={vi.fn()} />)
+    expect(screen.getByText('Which layout?')).toBeTruthy()
+    expect(screen.getByText('Single column')).toBeTruthy()
+    expect(screen.getByText('Two column')).toBeTruthy()
+    const rec = screen.getByTestId('decision-option-a')
+    expect(rec.className).toContain('recommended')
+  })
+
+  it('renders no iframe for a decision without visual', () => {
+    const { container } = render(<DecisionCard decision={pending} onAnswer={vi.fn()} />)
+    expect(container.querySelector('iframe')).toBeNull()
+  })
+
+  it('posts the chosen option and note on submit', () => {
+    const onAnswer = vi.fn()
+    render(<DecisionCard decision={pending} onAnswer={onAnswer} />)
+    fireEvent.change(screen.getByPlaceholderText(/optional note/i), {
+      target: { value: 'prefer a' },
+    })
+    fireEvent.click(screen.getByTestId('decision-submit'))
+    expect(onAnswer).toHaveBeenCalledWith(1, { optionId: 'a', note: 'prefer a' })
+  })
+
+  it('re-submits a different option on an answered decision', () => {
+    const onAnswer = vi.fn()
+    render(<DecisionCard decision={answered} onAnswer={onAnswer} />)
+    fireEvent.click(screen.getByTestId('decision-submit'))
+    expect(onAnswer).toHaveBeenCalledWith(2, { optionId: 'b', note: '' })
+  })
+})
diff --git a/skillgrid-ui/src/features/decisions/DecisionCard.tsx b/skillgrid-ui/src/features/decisions/DecisionCard.tsx
new file mode 100644
index 0000000..9e5dc22
--- /dev/null
+++ b/skillgrid-ui/src/features/decisions/DecisionCard.tsx
@@ -0,0 +1,165 @@
+import { useState } from 'react'
+
+// DecisionCard renders one decision from the inbox: the question, its
+// lettered options (the recommended one highlighted), an optional note, and a
+// Submit that hands {optionId, note} to onAnswer. On an answered decision it
+// shows the stored choice + note; the options stay re-answerable (the backend
+// appends a version) and re-selecting the picked option re-submits it.
+// A decision whose content carries a `visual` gets a sandboxed prototype
+// iframe in P3 — until then it renders a "visual available" placeholder,
+// never an iframe.
+
+export interface DecisionOption {
+  id: string
+  label: string
+  rationale?: string
+}
+
+export interface DecisionContent {
+  question: string
+  options: DecisionOption[]
+  recommended?: string
+  state: string
+  answeredOption?: string
+  answerNote?: string
+  updatedBy?: string
+  visual?: string
+}
+
+export interface Decision {
+  id: number
+  topicKey: string
+  title: string
+  createdAt: string
+  updatedAt: string
+  visibility: string
+  parseError?: boolean
+  content: DecisionContent
+}
+
+function day(iso: string): string {
+  const d = new Date(iso)
+  return isNaN(d.getTime()) ? iso : d.toISOString().slice(0, 10)
+}
+
+export function DecisionCard({
+  decision,
+  onAnswer,
+}: {
+  decision: Decision
+  onAnswer: (id: number, answer: { optionId: string; note: string }) => void
+}) {
+  const [note, setNote] = useState('')
+  const [selected, setSelected] = useState<string | null>(null)
+  const [busy, setBusy] = useState(false)
+
+  if (decision.parseError) {
+    return (
+      <div className="rounded-md border border-amber-500/40 bg-amber-500/10 p-3 text-sm text-amber-200">
+        Unreadable decision: {decision.topicKey}
+      </div>
+    )
+  }
+
+  const { content } = decision
+  const answered = content.state === 'answered'
+  const pickedId = answered ? content.answeredOption : undefined
+  const effective = selected ?? pickedId ?? content.recommended ?? ''
+  const hasVisual = Boolean(content.visual)
+
+  const submit = () => {
+    if (!effective || busy) return
+    setBusy(true)
+    onAnswer(decision.id, { optionId: effective, note })
+  }
+
+  return (
+    <div className="rounded-md border border-edge bg-card p-4">
+      <div className="mb-1 flex items-baseline justify-between gap-2">
+        <span className="truncate text-xs uppercase tracking-wide text-zinc-500">
+          {decision.topicKey}
+        </span>
+        <span className="shrink-0 text-[11px] text-zinc-500">{day(decision.createdAt)}</span>
+      </div>
+      <h3 className="mb-3 font-semibold text-zinc-100">{content.question}</h3>
+
+      <div className="flex flex-col gap-2" role="radiogroup" aria-label="Options">
+        {content.options.map((o, i) => {
+          const isRec = o.id === content.recommended
+          const isPicked = answered && o.id === content.answeredOption
+          const isEffective = effective === o.id
+          return (
+            <button
+              key={o.id}
+              type="button"
+              role="radio"
+              aria-checked={isEffective}
+              data-testid={`decision-option-${o.id}`}
+              disabled={busy}
+              onClick={() => setSelected(o.id)}
+              className={[
+                'flex items-start gap-2 rounded border p-2 text-left text-sm transition',
+                isRec ? 'recommended border-accent/70 bg-accent/10' : 'border-edge hover:bg-bg',
+                isEffective ? 'ring-1 ring-accent' : '',
+                isPicked ? 'border-emerald-500/60 bg-emerald-500/10' : '',
+                busy ? 'opacity-60' : '',
+              ].join(' ')}
+            >
+              <span className="mt-0.5 flex h-4 w-4 shrink-0 items-center justify-center rounded-full border border-zinc-500 text-[10px] font-semibold text-zinc-300">
+                {String.fromCharCode(65 + i)}
+              </span>
+              <span className="min-w-0 flex-1">
+                <span className="font-medium text-zinc-200">{o.label}</span>
+                {isRec && !answered && (
+                  <span className="ml-2 rounded bg-accent/20 px-1 py-0.5 text-[10px] uppercase tracking-wide text-accent">
+                    recommended
+                  </span>
+                )}
+                {isPicked && (
+                  <span className="ml-2 text-[11px] text-emerald-300">your choice</span>
+                )}
+                {o.rationale && (
+                  <span className="mt-0.5 block text-xs text-zinc-500">{o.rationale}</span>
+                )}
+              </span>
+            </button>
+          )
+        })}
+      </div>
+
+      {answered && content.answerNote && (
+        <div className="mt-2 text-xs text-zinc-500">
+          Note: <span className="text-zinc-300">{content.answerNote}</span>
+          {content.updatedBy && (
+            <span className="ml-2 text-zinc-600">· {content.updatedBy}</span>
+          )}
+        </div>
+      )}
+
+      {hasVisual && (
+        <div className="mt-3 rounded-md border border-dashed border-edge px-3 py-2 text-xs text-zinc-500">
+          Visual prototype available (renders in P3).
+        </div>
+      )}
+
+      <div className="mt-3 flex items-center gap-2">
+        <input
+          value={note}
+          onChange={(e) => setNote(e.target.value)}
+          placeholder="Optional note for the agent…"
+          aria-label="Optional note"
+          className="min-w-0 flex-1 rounded-md border border-edge bg-bg px-2 py-1 text-sm text-zinc-200 placeholder:text-zinc-600 focus:border-accent/60 focus:outline-none"
+        />
+        <button
+          type="button"
+          data-testid="decision-submit"
+          disabled={!effective || busy}
+          onClick={submit}
+          className="shrink-0 rounded-md bg-accent/80 px-3 py-1.5 text-xs font-medium text-zinc-950 hover:bg-accent disabled:opacity-40"
+        >
+          {answered ? 'Update answer' : 'Submit'}
+        </button>
+      </div>
+    </div>
+  )
+}
diff --git a/skillgrid-ui/src/features/decisions/DecisionsPage.test.tsx b/skillgrid-ui/src/features/decisions/DecisionsPage.test.tsx
new file mode 100644
index 0000000..197a4d4
--- /dev/null
+++ b/skillgrid-ui/src/features/decisions/DecisionsPage.test.tsx
@@ -0,0 +1,84 @@
+import { render, screen } from '@testing-library/react'
+import { describe, it, expect, vi, beforeEach, afterEach, type Mock } from 'vitest'
+import { DecisionsPage } from './DecisionsPage'
+import * as api from './api'
+
+// The page polls on a 4s interval. To test the interval without holding the
+// suite open, we spy on the global setInterval the component installs and
+// invoke the captured callback by hand (real timers, so findBy* waits work).
+function renderPolling() {
+  const intervalSpy = vi
+    .spyOn(globalThis, 'setInterval')
+    .mockImplementation(() => 1 as unknown as ReturnType<typeof setInterval>)
+  const utils = render(<DecisionsPage />)
+  const flush = () => {
+    for (const [arg] of intervalSpy.mock.calls as unknown as Array<[(cb: unknown) => void]>) {
+      if (typeof arg === 'function') (arg as () => void)()
+    }
+  }
+  return { ...utils, flush, intervalSpy }
+}
+
+let fetchSpy: Mock
+
+function row(id: number, state: 'pending' | 'answered'): api.DecisionRow {
+  return {
+    id,
+    topicKey: `demo/decision-${id}`,
+    title: `Decision ${id}`,
+    createdAt: '2026-09-19T10:00:00Z',
+    updatedAt: '2026-09-19T10:00:00Z',
+    visibility: 'team',
+    content: {
+      question: `Question ${id}?`,
+      options: [
+        { id: 'a', label: `Option A ${id}` },
+        { id: 'b', label: `Option B ${id}` },
+      ],
+      recommended: 'a',
+      state,
+      ...(state === 'answered' ? { answeredOption: 'b' } : {}),
+    },
+  }
+}
+
+describe('DecisionsPage', () => {
+  beforeEach(() => {
+    fetchSpy = vi.spyOn(api, 'fetchDecisions').mockResolvedValue([])
+    vi.spyOn(api, 'answerDecision').mockResolvedValue(undefined)
+  })
+  afterEach(() => {
+    vi.restoreAllMocks()
+  })
+
+  it('renders the pending decision inbox (question + options + created)', async () => {
+    fetchSpy.mockResolvedValue([row(1, 'pending')])
+    render(<DecisionsPage />)
+    expect(await screen.findByText('Question 1?')).toBeTruthy()
+    expect(screen.getByText('Option A 1')).toBeTruthy()
+    expect(screen.getByText('Option B 1')).toBeTruthy()
+    expect(screen.getByText(/2026-09-19/)).toBeTruthy()
+  })
+
+  it('fetches pending by default', async () => {
+    render(<DecisionsPage />)
+    await screen.findByText('No pending decisions.')
+    expect(fetchSpy).toHaveBeenCalledWith('pending')
+  })
+
+  it('refreshes the inbox on an interval', async () => {
+    fetchSpy.mockResolvedValue([row(1, 'pending')])
+    const { flush } = renderPolling()
+    await screen.findByText('Question 1?')
+    expect(fetchSpy).toHaveBeenCalledTimes(1)
+    flush()
+    expect(fetchSpy).toHaveBeenCalledTimes(2)
+  })
+
+  it('shows a stored answer for an answered decision', async () => {
+    fetchSpy.mockResolvedValue([row(2, 'answered')])
+    render(<DecisionsPage />)
+    await screen.findByText('Question 2?')
+    expect(screen.getByText(/your choice/i)).toBeTruthy()
+  })
+})
diff --git a/skillgrid-ui/src/features/decisions/DecisionsPage.tsx b/skillgrid-ui/src/features/decisions/DecisionsPage.tsx
new file mode 100644
index 0000000..da18d76
--- /dev/null
+++ b/skillgrid-ui/src/features/decisions/DecisionsPage.tsx
@@ -0,0 +1,86 @@
+import { useCallback, useEffect, useState } from 'react'
+import { DecisionCard, type Decision } from './DecisionCard'
+import { answerDecision, DecisionApiError, fetchDecisions } from './api'
+
+// DecisionsPage is the decision inbox: the pending interview questions the
+// agent posted, pollable on an interval (no WebSocket/SSE), with an answer
+// round-trip. A pending decision appears here, the user picks an option +
+// optional note and submits (POST …/answer), the card flips to state=answered
+// and shows the stored answer; a second submit re-answers (backend appends a
+// version). The default tab is the pending inbox.
+
+const REFRESH_MS = 4000
+
+type Tab = 'pending' | 'answered' | 'all'
+
+export function DecisionsPage() {
+  const [tab, setTab] = useState<Tab>('pending')
+  const [rows, setRows] = useState<Decision[]>([])
+  const [error, setError] = useState('')
+
+  const load = useCallback(async () => {
+    try {
+      setRows(await fetchDecisions(tab))
+      setError('')
+    } catch (e) {
+      setError(e instanceof DecisionApiError ? e.message : 'failed to load decisions')
+    }
+  }, [tab])
+
+  useEffect(() => {
+    void load()
+    const t = setInterval(() => void load(), REFRESH_MS)
+    return () => clearInterval(t)
+  }, [load])
+
+  const onAnswer = useCallback(
+    async (id: number, answer: { optionId: string; note: string }) => {
+      await answerDecision(id, answer)
+      void load()
+    },
+    [load],
+  )
+
+  return (
+    <div className="flex h-full flex-col">
+      <div className="flex items-center gap-3 border-b border-edge px-4 py-2">
+        <h1 className="text-sm font-semibold text-zinc-100">Decisions</h1>
+        <div className="flex gap-1" role="tablist" aria-label="Decision state">
+          {(['pending', 'answered', 'all'] as const).map((t) => (
+            <button
+              key={t}
+              type="button"
+              role="tab"
+              aria-selected={tab === t}
+              onClick={() => setTab(t)}
+              className={[
+                'rounded px-2.5 py-1 text-xs transition-colors',
+                tab === t ? 'bg-accent/20 text-accent' : 'text-zinc-500 hover:bg-card hover:text-zinc-300',
+              ].join(' ')}
+            >
+              {t}
+            </button>
+          ))}
+        </div>
+      </div>
+
+      {error && (
+        <div className="border-b border-red-500/30 bg-red-500/10 px-4 py-2 text-xs text-red-300">
+          {error}
+        </div>
+      )}
+
+      <div className="min-h-0 flex-1 overflow-y-auto p-4">
+        {rows.length === 0 ? (
+          <div className="text-sm text-zinc-500">No {tab} decisions.</div>
+        ) : (
+          <div className="mx-auto flex max-w-2xl flex-col gap-3">
+            {rows.map((d) => (
+              <DecisionCard key={d.id} decision={d} onAnswer={onAnswer} />
+            ))}
+          </div>
+        )}
+      </div>
+    </div>
+  )
+}
diff --git a/skillgrid-ui/src/features/decisions/api.ts b/skillgrid-ui/src/features/decisions/api.ts
new file mode 100644
index 0000000..e79cd8d
--- /dev/null
+++ b/skillgrid-ui/src/features/decisions/api.ts
@@ -0,0 +1,78 @@
+import { currentProjectName } from '../../lib/projects'
+import type { Decision } from './DecisionCard'
+
+// The decisions bridge read path is open; the answer (write) path is
+// token-gated when the server sets SKILLGRID_HTTP_TOKEN. The token comes from
+// the Settings view (localStorage) — the SPA cannot read the server's env, so
+// the user supplies it once and it is attached to the mutating request, exactly
+// like the tracker/kanban write routes.
+const TOKEN_KEY = 'skillgrid.httpToken'
+
+export class DecisionApiError extends Error {
+  status: number
+  constructor(message: string, status: number) {
+    super(message)
+    this.status = status
+  }
+}
+
+function getToken(): string {
+  try {
+    return window.localStorage.getItem(TOKEN_KEY) ?? ''
+  } catch {
+    return ''
+  }
+}
+
+function authHeaders(): Record<string, string> {
+  const token = getToken()
+  return token ? { Authorization: `Bearer ${token}` } : {}
+}
+
+async function parseError(res: Response): Promise<string> {
+  try {
+    const j = (await res.json()) as { error?: string }
+    if (j.error) return j.error
+  } catch {
+    /* keep statusText */
+  }
+  return res.statusText
+}
+
+// DecisionRow mirrors the backend GET /mnemonic/decisions row (camelCase JSON).
+export type DecisionRow = Decision
+
+let resolvedProject: string | null = null
+
+async function resolveProject(): Promise<string> {
+  if (resolvedProject) return resolvedProject
+  resolvedProject = await currentProjectName()
+  return resolvedProject
+}
+
+export async function fetchDecisions(state: string): Promise<DecisionRow[]> {
+  const project = await resolveProject()
+  const q = new URLSearchParams({ project })
+  if (state) q.set('state', state)
+  const res = await fetch(`/mnemonic/decisions?${q.toString()}`, {
+    headers: { Accept: 'application/json' },
+  })
+  if (!res.ok) throw new DecisionApiError(await parseError(res), res.status)
+  return (await res.json()) as DecisionRow[]
+}
+
+export async function answerDecision(
+  id: number,
+  answer: { optionId: string; note: string },
+): Promise<void> {
+  const project = await resolveProject()
+  const res = await fetch(
+    `/mnemonic/decisions/${id}/answer?project=${encodeURIComponent(project)}`,
+    {
+      method: 'POST',
+      headers: { 'Content-Type': 'application/json', Accept: 'application/json', ...authHeaders() },
+      body: JSON.stringify(answer),
+    },
+  )
+  if (!res.ok) throw new DecisionApiError(await parseError(res), res.status)
+}
diff --git a/skillgrid-ui/src/vite-env.d.ts b/skillgrid-ui/src/vite-env.d.ts
index 11f02fe..d3a8309 100644
--- a/skillgrid-ui/src/vite-env.d.ts
+++ b/skillgrid-ui/src/vite-env.d.ts
@@ -1 +1,2 @@
 /// <reference types="vite/client" />
+/// <reference types="vitest" />
diff --git a/skillgrid-ui/vite.config.ts b/skillgrid-ui/vite.config.ts
index bf09ccd..1d9fce0 100644
--- a/skillgrid-ui/vite.config.ts
+++ b/skillgrid-ui/vite.config.ts
@@ -1,22 +1,32 @@
+/// <reference types="vitest/config" />
 import { defineConfig } from 'vite'
 import react from '@vitejs/plugin-react'
 import tailwindcss from '@tailwindcss/vite'
 
 export default defineConfig({
   base: './',
   plugins: [react(), tailwindcss()],
   server: {
     proxy: {
       '/api': 'http://127.0.0.1:7438',
     },
   },
+  test: {
+    // Component tests (decisions view) render real DOM + act on it; the
+    // existing pure-logic tests keep running fine under jsdom too.
+    environment: 'jsdom',
+    // globals enables @testing-library/react's afterEach auto-cleanup (it
+    // looks the cleanup hook up via a global import), so each render() is
+    // torn down between tests.
+    globals: true,
+  },
   build: {
     outDir: '../skillgrid-cli/internal/mnemonic/http/ui/dist',
     emptyOutDir: true,
     // mermaid (~728 kB) is a single monolithic lib that Rollup can't split
     // further; it's dynamic-imported so it's lazy-loaded only when a diagram
     // renders, but it still trips the 500 kB warning. Bump the limit so the
     // build is quiet — the real win is per-library cacheable chunks, not
     // inlining heavy deps into the app bundle.
     chunkSizeWarningLimit: 800,
     rollupOptions: {
