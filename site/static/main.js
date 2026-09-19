// skillgrid site — vanilla JS for interactive parts. No framework.

// depth selector: T0..T3 tiers; click to activate, steps light up.
(function () {
  var grid = document.querySelector('[data-depth]');
  if (!grid) return;
  grid.addEventListener('click', function (e) {
    var t = e.target.closest('.tier');
    if (!t) return;
    grid.querySelectorAll('.tier').forEach(function (x) { x.classList.remove('active'); });
    t.classList.add('active');
    var idx = parseInt(t.dataset.tier, 10);
    grid.querySelectorAll('.tier').forEach(function (x) {
      var n = parseInt(x.dataset.tier, 10);
      x.querySelectorAll('.step').forEach(function (s) {
        var stepIdx = parseInt(s.dataset.step, 10);
        s.classList.toggle('on', stepIdx <= idx);
      });
    });
  });
})();

// copy buttons: .copy-btn[data-copy] copies adjacent code text
(function () {
  document.querySelectorAll('.copy-btn').forEach(function (btn) {
    btn.addEventListener('click', function () {
      var code = btn.getAttribute('data-copy');
      if (!code && btn.parentElement) {
        var el = btn.parentElement.querySelector('code');
        if (el) code = el.textContent.trim();
      }
      if (!code) return;
      var done = function () {
        var old = btn.textContent;
        btn.textContent = 'Copied';
        setTimeout(function () { btn.textContent = old; }, 1400);
      };
      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(code).then(done, function () { fallback(); });
      } else { fallback(); }
      function fallback() {
        var ta = document.createElement('textarea');
        ta.value = code; document.body.appendChild(ta); ta.select();
        try { document.execCommand('copy'); } catch (e) {}
        document.body.removeChild(ta); done();
      }
    });
  });
})();

// install tabs: .tabs .tab[data-pane] toggles .code-row[data-pane]
(function () {
  document.querySelectorAll('.tabs').forEach(function (tabs) {
    tabs.addEventListener('click', function (e) {
      var t = e.target.closest('.tab');
      if (!t) return;
      var scope = t.closest('[data-install]') || t.parentElement.parentElement;
      tabs.querySelectorAll('.tab').forEach(function (x) { x.classList.remove('active'); });
      t.classList.add('active');
      var pane = t.dataset.pane;
      scope.querySelectorAll('.code-row').forEach(function (r) {
        r.style.display = r.dataset.pane === pane ? 'flex' : 'none';
      });
    });
  });
})();

// mermaid: render .mermaid blocks (ESM CDN, one diagram at a time)
(function () {
  if (!document.querySelector('pre.mermaid')) return;
  function render() {
    import('https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.esm.min.mjs').then(function (m) {
      var mermaid = m.default;
      mermaid.initialize({
        startOnLoad: false,
        theme: 'dark',
        themeVariables: {
          primaryColor: '#050709', primaryTextColor: '#ffffff',
          primaryBorderColor: '#2e3238', lineColor: '#6ee7a0',
          secondaryColor: '#111214', tertiaryColor: '#020013',
          mainBkg: '#050709', nodeBorder: '#2e3238',
          clusterBkg: '#111214', clusterBorder: '#2e3238',
          titleColor: '#ffffff', edgeLabelBackground: '#050709'
        }
      });
      var nodes = document.querySelectorAll('pre.mermaid');
      var p = Promise.resolve();
      for (var i = 0; i < nodes.length; i++) {
        (function (n) {
          p = p.then(function () {
            return mermaid.run({ nodes: [n] }).catch(function (e) {
              console.error('[skillgrid] mermaid diagram failed:', e && e.message ? e.message : e);
            });
          });
        })(nodes[i]);
      }
    });
  }
  if (document.readyState === 'complete') render();
  else window.addEventListener('load', render);
})();

// pipeline: no JS needed (static flex), but smooth-scroll for #anchor nav
(function () {
  document.querySelectorAll('a[href^="#"]').forEach(function (a) {
    a.addEventListener('click', function (e) {
      var id = a.getAttribute('href').slice(1);
      var el = document.getElementById(id);
      if (el) { e.preventDefault(); el.scrollIntoView({ behavior: 'smooth', block: 'start' }); }
    });
  });
})();
