# Native read modules

Each domain owns a module.js, manifest.json, tests and evidence report.
Only explicitly registered, schema-verified, read-only operations are loaded.
The shared runtime invokes request(input), calls the declared SDK command and
passes its response data through project(response). Manifests describe inputs;
projections must exclude credentials. Local verification is not live proof.
