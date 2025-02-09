import React, { useState, useEffect } from 'react';
import './VersionDropdown.css';

function VersionDropdown() {
  console.log("Navbar Version dropdown initialized");

  const githubApiUrl = 'https://api.github.com/repos/akuity/kargo/branches?protected=true';
  const protocol = 'https://';
  const domain = 'docs.kargo.io';
  const latestVersionLabel = 'Latest Version';
  const latestVersionUrl = `${protocol}${domain}`;
  const edgeVersionLabel = 'Edge Version (main)';
  const edgeVersionUrl = `${protocol}main.${domain}`;
  const unrecognizedVersionLabel = 'Unrecognized Version';

  const [versions, setVersions] = useState([]);
  const [loading, setLoading] = useState(true);
  const [currentVersion, setCurrentVersion] = useState('');

  // Change the implementation of the currentUrl function to aid in testing
  const currentUrl = () => new URL(window.location.href);

  const versionLabel = (major, minor) => `v${major}.${minor}`;

  const fetchVersions = async () => {
    try {
      console.log("Before fetching versions");
      const response = await fetch(githubApiUrl);
      const branches = await response.json();
      console.log("Fetched branches are: ", branches);

      const releaseBranches = branches
        .map(branch => branch.name)
        .filter(name => /^release-([1-9]\d*\.\d+)$/.test(name))
        .map(name => {
          const [major, minor] = name.replace('release-', '').split('.');
          return {
            version: versionLabel(major, minor),
            url: `${protocol}${name.replace('.', '-')}.${domain}`
          };
        });

      console.log("These are release branches before sorting and updating 0 element: ", releaseBranches);
      releaseBranches.sort((a, b) => {
        if (a.version > b.version) {
          return -1;
        }
        if (a.version < b.version) {
          return 1;
        }
        return 0;
      });
      // Overwrite the first element with the latest version
      releaseBranches[0] = {
        version: latestVersionLabel,
        url: latestVersionUrl
      };
      // Put the "edge" version at the end of the list
      releaseBranches.push({
        version: edgeVersionLabel,
        url: edgeVersionUrl,
      })
      const currentVersion = getCurrentVersion();
      // If the current version is not recognized, add it to the end of the list
      if (currentVersion === unrecognizedVersionLabel) {
        const url = currentUrl();
        releaseBranches.push({
          version: unrecognizedVersionLabel,
          url: `${url.protocol}//${url.hostname}${url.port ? `:${url.port}` : ''}`
        });
      }
      console.log("These are release branches: ", releaseBranches);
      setVersions(releaseBranches);
      setLoading(false);
    } catch (error) {
      console.error('Error fetching versions:', error);
      setLoading(false);
    }
  };

  const getCurrentVersion = () => {
    const url = currentUrl();
    if (url.hostname === new URL(latestVersionUrl).hostname && !url.port) {
      return 'Latest Version';
    }
    if (url.hostname === new URL(edgeVersionUrl).hostname && !url.port) {
      return 'Edge Version (main)';
    }
    const match = url.hostname.match(/^release-(\d+)-(\d+)\.docs\.kargo\.io$/);
    return match && !url.port ? versionLabel(match[1], match[2]) : unrecognizedVersionLabel;
  };

  useEffect(() => {
    setCurrentVersion(getCurrentVersion());
    fetchVersions();
  }, []);

  const handleVersionChange = (event) => {
    const selectedVersion = versions.find(v => v.version === event.target.value);
    if (selectedVersion) {
      window.location.href = selectedVersion.url;
    }
  };

  if (loading) return null;

  return (
    <select
      className="version_dropdown"
      onChange={handleVersionChange}
      value={currentVersion}
    >
      {versions.map(version => (
        <option key={version.version} value={version.version}>
          {version.version}
        </option>
      ))}
  </select>
  );
}

export default VersionDropdown;
