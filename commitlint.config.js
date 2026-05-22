module.exports = {
  extends: ['@commitlint/config-conventional'],
  rules: {
    'scope-enum': [2, 'always', [
      'estimator', 'monitor', 'storage',
      'analysis', 'infra', 'docs', 'ci'
    ]],
    'subject-max-length': [2, 'always', 72],
    'subject-case': [2, 'always', 'lower-case'],
  }
}
