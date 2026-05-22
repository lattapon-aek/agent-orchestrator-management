# AOM

AOM is a project-local control plane for native CLI agents.

## Version

```bash
aom version
aom --version
```

## Install

Use the CLI when you are inside a repo checkout:

```bash
aom install
aom install --test
aom install --dry
```

Or use the repo script directly:

```bash
./scripts/install.sh
./scripts/install.sh --test
./scripts/install.sh --dry
```

## Update

Use the CLI when you are inside a repo checkout:

```bash
aom update
aom update --test
```

Or use the repo script directly:

```bash
./scripts/update.sh
./scripts/update.sh --test
```

## Uninstall

Use the CLI:

```bash
aom uninstall
sudo aom uninstall
```

Or use the wrapper script:

```bash
./scripts/uninstall.sh
```

## Project docs

- `docs/AOM-planning.md`
- `docs/AOM-milestones.md`
- `docs/project-structure.md`
- `docs/engineering-guidelines.md`

