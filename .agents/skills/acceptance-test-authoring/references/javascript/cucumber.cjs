const { extractAll } = require('./extract-gherkin.cjs');
const { effectivePaths } = require('./skillgrid-paths.cjs');

extractAll();

module.exports = {
  default: {
    paths: effectivePaths(),
    import: ['support/**/*.js', 'step-definitions/**/*.js'],
    format: ['progress-bar', ['html', 'reports/cucumber-report.html']],
  },
};
