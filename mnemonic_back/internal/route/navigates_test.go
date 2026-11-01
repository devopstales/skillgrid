package route

import (
	"testing"
)

// routerFixture returns the fixture source for a router framework.
func routerFixture(framework string) (file, src string) {
	switch framework {
	case "nextjs":
		return "page.tsx", `import { useRouter } from 'next/router';

export function Page() {
  const router = useRouter();
  return (
    <button onClick={() => router.push('/about')}>About</button>
  );
}

export function Nav() {
  return <Link href="/settings">Settings</Link>;
}
`
	case "react-router":
		return "App.tsx", `import { Routes, Route, useNavigate } from 'react-router-dom';

export function App() {
  const navigate = useNavigate();
  return (
    <>
      <Routes>
        <Route path="/home" element={<Home />} />
      </Routes>
      <button onClick={() => navigate('/profile')}>Profile</button>
    </>
  );
}
`
	case "sveltekit":
		return "page.svelte", `import { goto } from '$app/navigation';

<script>
  function home() { goto('/'); }
</script>

<button on:click={home}>Home</button>
`
	case "vue-router":
		return "main.js", `import router from './router';

function go() {
  router.push({ name: 'settings' });
}

function goPath() {
  router.push('/about');
}
`
	}
	return "", ""
}

// TestNavigatesForRouters covers @step-01 (Scenario: Router navigations
// produce navigates edges): each router fixture yields navigates edges from
// the sending function to the named screen; literal destinations are
// EXTRACTED, markup-written links are INFERRED.
func TestNavigatesForRouters(t *testing.T) {
	cases := []struct {
		framework string
		// want screens that must appear, with their expected confidence.
		want []struct {
			screen string
			conf   string
		}
	}{
		{"nextjs", []struct {
			screen string
			conf   string
		}{
			{"/about", ConfidenceExtracted},
			{"/settings", ConfidenceInferred},
		}},
		{"react-router", []struct {
			screen string
			conf   string
		}{
			{"/home", ConfidenceInferred},
			{"/profile", ConfidenceExtracted},
		}},
		{"sveltekit", []struct {
			screen string
			conf   string
		}{
			{"/", ConfidenceExtracted},
		}},
		{"vue-router", []struct {
			screen string
			conf   string
		}{
			{"/settings", ConfidenceExtracted},
			{"/about", ConfidenceExtracted},
		}},
	}
	for _, tc := range cases {
		t.Run(tc.framework, func(t *testing.T) {
			file, src := routerFixture(tc.framework)
			b := Build(file, []byte(src), fileSyms("App", "Page"), known())
			if len(b.Navs) == 0 {
				t.Fatalf("%s: expected navigates edges, got none (src: %s)", tc.framework, src)
			}
			for _, w := range tc.want {
				found := false
				for _, n := range b.Navs {
					if n.ToName == w.screen {
						found = true
						if n.Confidence != w.conf {
							t.Errorf("%s navigates to %s: confidence = %q, want %q", tc.framework, w.screen, n.Confidence, w.conf)
						}
						if n.FromSymbol == 0 {
							t.Errorf("%s navigates to %s: missing sending function (FromSymbol=0)", tc.framework, w.screen)
						}
					}
				}
				if !found {
					t.Errorf("%s: no navigates edge to screen %q (got: %v)", tc.framework, w.screen, navScreens(b.Navs))
				}
			}
		})
	}
}

// navScreens returns the sorted screen names of a nav set (for failure output).
func navScreens(navs []NavigationNode) []string {
	out := make([]string, 0, len(navs))
	for _, n := range navs {
		out = append(out, n.ToName)
	}
	return out
}

// TestNavigatesMarkupInferred covers @step-01 (markup-written links are
// INFERRED, programmatic literals are EXTRACTED): a <Link href> produces an
// INFERRED navigates edge while router.push('/x') produces an EXTRACTED one.
func TestNavigatesMarkupInferred(t *testing.T) {
	_, src := routerFixture("nextjs")
	b := Build("page.tsx", []byte(src), fileSyms("Page", "Nav"), known())
	conf := map[string]string{}
	for _, n := range b.Navs {
		conf[n.ToName] = n.Confidence
	}
	if conf["/about"] != ConfidenceExtracted {
		t.Errorf("router.push('/about') should be EXTRACTED, got %q", conf["/about"])
	}
	if conf["/settings"] != ConfidenceInferred {
		t.Errorf("<Link href='/settings'> should be INFERRED, got %q", conf["/settings"])
	}
}

// TestNavigatesNoRouterProducesNone covers the no-routes case: a file with no
// router patterns yields no navigates edges and no route nodes.
func TestNavigatesNoRouterProducesNone(t *testing.T) {
	b := Build("plain.js", []byte("function foo() {\n  return 1;\n}\n"), fileSyms("foo"), known())
	if len(b.Navs) != 0 {
		t.Errorf("expected no navigates edges for a plain file, got %d", len(b.Navs))
	}
	if len(b.Nodes) != 0 {
		t.Errorf("expected no route nodes for a plain file, got %d", len(b.Nodes))
	}
}
