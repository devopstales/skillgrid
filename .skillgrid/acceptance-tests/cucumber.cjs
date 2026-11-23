const path = require('path');

const { extractAll } = require('./extract-gherkin.cjs');
const { effectivePaths } = require('./skillgrid-paths.cjs');

extractAll();

module.exports = {
  default: {
    paths: effectivePaths(),
    import: ['support/**/*.js', 'step-definitions/**/*.js'],
    format: ['progress-bar', ['html', path.join(__dirname, 'reports', 'cucumber-report.html')]],
  },
};
