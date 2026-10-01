/**
 * The two resources a deep link can be attached to. Each maps to its own field
 * on ProjectConfig/ClusterConfig (`freightLinks`, `stageLinks`) and brings its
 * own evaluation context, so the forms are parameterized by kind rather than
 * duplicated.
 *
 * Every example below is one that survives a resource with none of its
 * optional fields set. That matters more than it looks: the context is the
 * resource serialized to JSON, so an `omitempty` field that is unset is not
 * null. It is absent, and `index`, `len()` or `[...]` on an absent field
 * raises an error that makes the whole link disappear with no explanation.
 * `get` and `?.` return empty instead, so they are what the examples teach.
 */
export type DeepLinkKind = 'freight' | 'stage';

type DeepLinkKindInfo = {
  /** Card title. */
  title: string;
  /** How the resource is named in prose. */
  resource: string;
  /** How the whole set of them is named in prose. Freight is a mass noun. */
  allResources: string;
  /** Root of the Go template context, as it is written in a URL template. */
  templateRoot: string;
  /** Root of the expression context, as it is written in a condition. */
  expressionRoot: string;
  /** Where links of this kind surface in the UI. */
  shownAt: string;
  urlExamples: string[];
  conditionExamples: string[];
};

export const deepLinkKinds: Record<DeepLinkKind, DeepLinkKindInfo> = {
  freight: {
    title: 'Freight Links',
    resource: 'Freight',
    allResources: 'every piece of Freight',
    templateRoot: '.freight',
    expressionRoot: 'freight',
    shownAt: 'the Links menu of a piece of Freight',
    urlExamples: [
      '{{ .freight.metadata.name }}',
      '{{ .freight.origin.name }}',
      '{{ get .freight "alias" }}'
    ],
    conditionExamples: ['freight.images != nil', 'freight.origin.name == "my-warehouse"']
  },
  stage: {
    title: 'Stage Links',
    resource: 'Stage',
    allResources: 'every Stage',
    templateRoot: '.stage',
    expressionRoot: 'stage',
    shownAt: 'the Links menu of a Stage',
    urlExamples: [
      '{{ .stage.metadata.name }}',
      '{{ .stage.metadata.namespace }}',
      '{{ get .stage.metadata.labels "team" }}'
    ],
    conditionExamples: ['stage.spec.verification != nil', 'stage.metadata.labels?.env == "prod"']
  }
};
