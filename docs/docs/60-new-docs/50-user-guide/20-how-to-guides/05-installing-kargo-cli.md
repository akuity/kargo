---
sidebar_label: Installing the CLI
---

# Installing the Kargo CLI

The Kargo CLI provides a command-line interface to manage Kargo efficiently.

The simplest and recommended way to install the CLI is via the <Hlt>CLI Tab</Hlt> in the Kargo UI.
This ensures the Kargo CLI version matches the Kargo version installed in your cluster,
making it easier to maintain compatibility and avoid issues.

![CLI Tab in Kargo UI](./img/cli-installation.png)

_Alternatively_, if you prefer to install the CLI manually, you can run the following command:

<Tabs groupId="os">
<TabItem value="mac-linux-wsl" label="Mac, Linux, or WSL" default>

```shell
arch=$(uname -m)
[ "$arch" = "x86_64" ] && arch=amd64
curl -L -o kargo https://github.com/akuity/kargo/releases/latest/download/kargo-"$(uname -s | tr '[:upper:]' '[:lower:]')-${arch}"
chmod +x kargo
```

</TabItem>
<TabItem value="windows" label="Windows Powershell">

```shell
Invoke-WebRequest -URI https://github.com/akuity/kargo/releases/latest/download/kargo-windows-amd64.exe -OutFile kargo.exe
```

</TabItem>
</Tabs>

After downloading, move the `kargo` binary (for Mac, Linux, or WSL) or `kargo.exe` (for Windows) to a
folder included in your `PATH` environment variable. This makes it accessible from anywhere in the terminal.