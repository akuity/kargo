import { IconDefinition } from '@fortawesome/fontawesome-svg-core';
import {
  faBitbucket,
  faDocker,
  faGitAlt,
  faGithub,
  faGitlab,
  faJira,
  faAtlassian,
  faSlack,
  faRedhat,
  faGit,
  faJenkins,
  faDigitalOcean,
  faGithubAlt
} from '@fortawesome/free-brands-svg-icons';
import {
  faArrowUpRightFromSquare,
  faBox,
  faChartLine,
  faCheckCircle,
  faCircleNodes,
  faCode,
  faDatabase,
  faExternalLink,
  faFileCode,
  faGlobe,
  faLink,
  faServer,
  faTag
} from '@fortawesome/free-solid-svg-icons';

const ICONS: Record<string, IconDefinition> = {
  // brand
  faBitbucket,
  faDocker,
  faGitAlt,
  faGithub,
  faGitlab,
  faJira,
  faSlack,
  faAtlassian,
  faRedhat,
  faGit,
  faJenkins,
  faDigitalOcean,
  faGithubAlt,
  // solid
  faArrowUpRightFromSquare,
  faBox,
  faChartLine,
  faCheckCircle,
  faCircleNodes,
  faCode,
  faDatabase,
  faFileCode,
  faGlobe,
  faLink,
  faServer,
  faTag
};

export function iconExists(name?: string): boolean {
  return !!name && name in ICONS;
}

export function resolveIcon(name?: string): IconDefinition {
  return (name && ICONS[name]) || faExternalLink;
}
