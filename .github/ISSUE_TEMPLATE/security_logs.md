---
name: Security log
about: Propose adding security-related logs or tagging existing logs with security fields
title: "seclog: [Event Description]"
labels: security-log
assignees: notfromstatefarm
---
# Event to be logged

Specify the event that needs to be logged or existing logs that need to be tagged.

# Proposed level

What security level should these events be logged under? 

Security-related logs are tagged with a `security` field to make them easier to find, analyze, and report on.

| Level | Friendly Level | Description                                                                                       | Example                                     |
|-------|----------------|---------------------------------------------------------------------------------------------------|---------------------------------------------|
| 1     | Low            | Unexceptional, non-malicious events                                                               | Successful access                           |
| 2     | Medium         | Could indicate malicious events, but has a high likelihood of being user/system error             | Access denied                               |
| 3     | High           | Likely malicious events but one that had no side effects or was blocked                           | Out of bounds symlinks in repo              |
| 4     | Critical       | Any malicious or exploitable event that had a side effect                                         | Secrets being left behind on the filesystem |
| 5     | Emergency      | Unmistakably malicious events that should NEVER occur accidentally and indicates an active attack | Brute forcing of accounts                   |


# Common Weakness Enumeration

Is there an associated [CWE](https://cwe.mitre.org/) that could be tagged as well?
