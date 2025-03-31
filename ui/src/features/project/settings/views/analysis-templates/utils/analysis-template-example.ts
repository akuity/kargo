export const getAnalysisTemplateYAMLExample = (namespace: string) =>
  `apiVersion: argoproj.io/v1alpha1
kind: AnalysisTemplate
metadata:
  name: error-rate
  namespace: ${namespace}
spec:
  args:
  - name: service-name
  metrics:
  - name: error-rate
    interval: 5m
    successCondition: result <= 0.01
    failureLimit: 3
    provider:
      datadog:
        apiVersion: v2
        interval: 5m
        query: |
          sum:requests.error.rate{service:{{args.service-name}}}`;
