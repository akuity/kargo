# Akuity Documentation Style Guide

This style guide is a compendium of general guidelines that Akuity documentation
contributors should attempt to follow in order to ensure proper formatting,
consistent phrasing, and maximized readability throughout.

Some general things to keep in mind:

* Simple language is best. Many readers may speak English as a second language.
  It is best to avoid large vocabulary words or idioms that may be familiar only
  to native English speakers.

* Some people don't like to read. Others haven't got the time. To make the most
  of whatever time the reader is willing to spend on our docs, remember there is
  an art to conveying more information with fewer words.

* Consistency is a virtue. If our documentation is not internally consistent,
  some readers may consciously notice, but far worse, those who don't notice may
  simply end up confused.

* Whether consciously or subconsciously, readers take cues from how things are
  formatted. These might be obvious things like monospaced text (`like this`)
  hinting that something is a command, but can also be far more subtle. For
  example, judicious use of whitespace can make text feel less "cramped" and
  this can subconsciously affect a reader's feelings by making a sequence of
  instructions they should follow seem less intimidating. It can be the
  difference between overwhelming the reader or not.

* The quality of our documentation reflects the quality of our platform. High
  quality docs instill confidence in the platform, while poor quality docs can
  lead readers, consciously or otherwise, to question the quality of our
  platform.

* Above all else, we want to make the docs _feel_ approachable.

* Nothing here is a hard and fast rule. These are _guidelines_; not laws. There
  will be reasons to ignore some of these guidelines at times. Use your
  judgement.

## Phraseology

This is a list of some preferred word choices that we have arrived at by
consensus.

* **The Akuity Platform:** This is the correct way to refer to our product, as
  opposed to "Akuity Platform" (without the definite article "the"). Both
  "Akuity" _and_ "Platform" should always be capitalized. "The" should only be
  capitalized at the start of a sentence. i.e. The following is _wrong_: "You
  will love working with The Akuity Platform." Do not capitalize the word
  "platform" when _not_ pairing it with "Akuity."

* **The Akuity Agent:** This is such an integral component of our platform that
  we electively capitalize "Agent" in "Akuity Agent." "The" should only be
  capitalized at the start of a sentence. Do not capitalize the word
  "agent" when _not_ pairing it with "Akuity."

* **Dashboard:** Prefer "dashboard" over "the UI," where you are able. These
  terms vary in specificity. _The_ dashboard is _a_ UI. If the UI you want to
  talk about is the Akuity Platform's dashboard, _say_ "dashboard." We do not capitalize this word unless it begins a sentence.

* **Click:** We instruct users to _click_ buttons rather than "press" (or "tap,"
  "select," etc.) For non-button UI elements like a tab, checkbox, or radio
  button, alternatives like "select" are still be appropriate.

* TODO: There will undoubtedly be many additions to this list that haven't been 
  discovered yet. Add them as you find them!

## Text conventions

### Capitalization

This can be deceptively challenging.

In general, follow typical English rules for capitalizing words -- i.e.
Sentences should begin with a capitalized word and proper nouns should be
capitalized. There are, however, many exceptions to this in technical writing.

* In general, capitalize the first word of a list item, even if the list item
  is not a complete sentence or is the second half of a complete sentence that
  was started by some introductory text above the list.

* Many products, projects, companies, etc. may, as a matter of branding, _not_
  begin their proper name with a capital letter and that choice should be
  respected, _even when that word begins a sentence or a list item_. For
  example, do _not_ capitalize words like "iPhone" or "minikube," even at the
  start of a sentence. If ever in doubt, consult a project or company's own
  documentation to understand the conventions they use for their own name.

* It is often tempting, and sometimes appropriate to capitalize words that are
  of particular importance, even if they would not ordinarily be capitalized.
  This should be done sparingly, however, because if it is overused, it begins
  to look _random_. If you think a word that isn't ordinarily capitalized is
  important enough to warrant it, consider searching the existing documentation
  first to see if there is already a clear convention for that word.

* There is no universally accepted rule for this. It is sometimes tempting to
  treat headings like _titles_ and capitalize every word, but as a matter of
  style, we do not do this. We capitalize the first word of a heading (any
  level) and not the rest, except for words that should be capitalized per
  previous guidelines.

### Monospaced text

Reserve the use of monospaced text (`like this`) only for highly technical words
like type names, environment variables, or things a reader may explicitly wish
to copy/paste or type, such as code snippets or CLI commands.

Examples of when to use:

* You can configure Argo CD notifications using Kubernetes `ConfigMap` or
  `Secret` resources.

* Type `kubectl get pods --all-namespaces`.

Examples of when _not_ to use (these are _wrong_):

* Before getting started, you should be familiar with `Kubernetes`.

* Find the URL for your `git` repository. (This is a tricky one, but in this
  context, we're talking about Git as _thing_ and _not_ the `git` _command_. The
  correct way to write this would be "Git.")

Do _not_ use monospaced text when referencing UI elements like buttons or form
field names. Refer to the next section for a superior option.

### Referencing UI elements

When referencing UI elements, like buttons or form field names, you should
enclose the applicable text within our custom `<hlt>` tag.

For example:

```markdown
1. Do this.
1. Then do this.
1. Click <hlt>Save</hlt>.
```

* If a UI element contains additional, _simple_ characters, include them within
  the highlight. For example, `<hlt>+ Add</hlt>` is correct if that is what the
  button you are referencing actually says. For more exotic symbols that are
  more difficult to type, such as an icon/emoji, do _not_ include them.

* If a UI element's text varies because it is user-specific, for instance, use
  italics _within_ the enclosing `<hlt>` tag to denote this. In general, this is
  easily interpreted by readers. For example: `<hlt>_your instance name_</hlt>`.

* Avoid explicitly saying "button" wherever it isn't needed for clarity. For
  instance, "click submit" is preferred over "click the submit button." The
  difference is subtle, but when viewed in the context of a list of
  instructions, less text is psychologically less intimidating to the reader.
  _Do_ qualify other UI elements that are not buttons. "Select the Organizations
  tab" is appropriate.

### Arrows

There are places in text, for instance when referencing the "breadcrumbs" a
reader should follow to arrive at a particular screen in the dashboard, where it
will be tempting to utilize greater than signs (&gt;) to denote some form of
hierarchy. _**If you are not careful, these characters can cause big problems
with rendering the markdown properly.**_ As a matter of convention, we have
settled on using arrows (→) in these cases instead. On a Mac, you can type that
character by pressing `ctrl` + `cmd` + `space` all at once and searching for
"rightwards arrow."

### Emojis

Supplemental _information_ should be conveyed to readers using conventions
discussed in the next section, so emoji like "⚠️" or "⛔️" do not have much use
in our documentation, but on _rare_ occasions, there are opportunities to
connect with a reader on a more emotional level using emoji that call attention
to a _feeling_ that our text may have evoked. As a humorous example, "😜" might
convey that the author is acutely aware of how weirdly the preceding statement
may have registered with the reader. It's an opportunity to find common ground
with the reader and establish a rapport, **but do not over-use this!**

## Supplemental information

It can be very helpful to offset important information from the rest of your
text. Although this is not standard markdown, Docusaurus enables you do this
using syntax such as the following:

```markdown
:::note

This is something you should take note of!

:::
```

Apart from `note` (gray), you can also use `tip` (green), `info` (blue),
`caution` (yellow), and `danger` (red).

> ⚠️&nbsp;**Never include whitespace between `:::` and `note`/`tip`, etc. or it will not render correctly!**

Here are some guidelines on when to use each:

* If something is important, but not unusually critical, _don't_ offset the text
  at all. Readers who are in a hurry may mentally dismiss a gray, green, or blue
  box as unimportant.

* If something is worth knowing, but it won't harm the user if they were to
  overlook it, break it out into a box with a non-threatening color like gray,
  green or blue.

* If something is critical to know and must not be overlooked, use a color like
  yellow or red to signal its importance.

## List conventions

Use lists wherever you can. Lists structure information _for_ the reader instead
of requiring them to parse and mentally organize information that's been
presented in paragraph form. It is, for instance, very easy to miss a step in,
"Do this first, then this, and then this," when compared to the same
instructions presented as a list.

Lists are one of the hardest things to keep consistent, but following these
guidelines will help.

* Use bulleted lists when order doesn't matter. Good examples would be cases
  where a set of prerequisites is listed or cases where the reader is presented
  with _choices_.

* Use enumerated lists when order matters. The most common case for this is a
  sequence of instructions wherein one step must be completed before moving to
  the next.

    * When using an enumerated list to express instructions, avoid putting
      preconditions in the first list item. To keep the list concise and
      readable, put this type of information in some introductory text instead.

      For example, instead of this:

      ```markdown
      1. After you have registered for the platform and activated your account, do this.

      1. Then do this.
      ```

      Consider this:

      ```markdown
      After you have registered for the platform and activated your account:

      1. Do this.

      1. Then do this.
      ```

    * Start all lines with the number `1`. Docusaurus, like most markdown
      renderers, will enumerate the list properly _for you_. By leading every
      line with `1`, you will avoid anyone having to re-numerate the list
      manually if and when the list is edited in the future. More importantly,
      you will eliminate the possibility of human error that can arise from
      manually re-numerating the list.

* It is ok to mix ordered and unordered lists! For instance, if you were
  defining a set of instructions and subordinate to one of those steps, the user
  has _options_ and must pick _one_, the following approach is very clear:

  ```markdown
  1. Do this first.

  1. Second, you can either:

      * Do it this way or

      * Do it this way.

  1. Do this third.
  ```

* _Always_ include blank lines between list items, as in the example above.
  Docusaurus renders lists that include blank lines between items differently
  from lists that do not. For consistency, we need to stick with one approach.
  The approach that uses extra space seems like a good choice because the lists
  look less "dense" or "cramped" when rendered and read, which makes them feel
  more approachable.

* Be mindful of terminating punctuation like periods or question marks in your
  lists. If list items are complete sentences or if list items _complete_ a
  sentence that was started by some introductory text, then terminal punctuation
  should be used. Otherwise, terminal punctuations should be omitted.

  In the following example, every list item is a complete sentence and should have
  terminating punctuation:

  ```markdown
  Follow these steps:

  1. Do this first.

  1. Do this second.

  1. Do this third.
  ```

  In the next example, each list item completes a sentence that was started by
  some introductory text:

  ```markdown
  You may enjoy the Akuity Platform if you...

  * Already love Argo CD.

  * Don't want to host or manage Argo CD yourself.
  ```

  In the last example, each list item is neither a complete sentence, nor does it
  complete a sentence that was started by some introductory text.

  ```markdown
  The documentation assumes basic familiarity with the following:

  * Kubernetes

  * GitOps

  * Argo CD
  ```

## Instructions that vary

In a few places throughout the documentation, instructions for the readers may
vary according to their operating system. Similarly, there will be cases where
we wish to present different instructions for users of the dashboard, CLI, and
API.

In these cases, please make use of _tabs_ to prevent mutually exclusive options
from taking up space and becoming a "wall of text."

> ⚠️&nbsp;**To use tabs, your file extension must be `.mdx` instead of `.md`.**

Here is an example of using tabs correctly:

```markdown
<Tabs groupId="os">

<TabItem value="mac" label="Mac" default>
Instructions for Mac...
</TabItem>

<TabItem value="linux" label="Linux">
Instructions for Linux...
</TabItem>

<TabItem value="windows" label="Windows">
Instructions for Windows...
</TabItem>

</Tabs>
```

## Managing the doc tree

Docusaurus offers a few options for organizing the doc tree:

⛔️ One approach involves assigning weights in a document's front matter like so,
**but please do not do this**:

```yaml
---
sidebar_position: 5
---
```

🟢 Instead, prefix file and folder names with a weight and a dash. Docusaurus
can infer the correct order of documents and sections from this naming
convention. (It will not impact the URLs.) The main benefit of this approach is
the order files are displayed in your text editor will correctly reflect the
order of the rendered doc tree.

The following tree illustrates the approach:

```
.
├── 10-akuity-platform
│   ├── 10-portal.md
│   ├── 20-architecture.md
│   ├── 30-agent.md
│   └── index.md
├── 20-getting-started
│   ├── 10-create-argo-cd-instance.md
│   ├── 20-connect-kubernetes-cluster.md
│   ├── 30-configure-admin-user.md
│   ├── 40-access-argo-cd-instance.mdx
│   └── _category_.json
├── 30-how-to-guides
│   ├── 10-changing-contexts.md
│   ├── 20-upgrading-argo-cd.md
│   ├── 30-enabling-notifications.md
│   ├── 40-using-webhooks.md
│   ├── 50-enabling-external-access.md
│   ├── 60-managing-system-accounts.md
│   └── _category_.json
├── 40-changelog.md
└── 50-faq.mdx
```

🟢 Also avoid using consecutive numbers as weights. The doc tree was initially
created with documents and section weighted at intervals of 10 **to create the
possibility of inserting new sections or documents later without having to
renumber everything.** As a rule of thumb, when inserting a _new_ section or
document, assign it a weight that is halfway between the weights of the
preceding and following documents or sections. This _preserves_ the possibility
of inserting still more documents or sections in the future without renumbering.
