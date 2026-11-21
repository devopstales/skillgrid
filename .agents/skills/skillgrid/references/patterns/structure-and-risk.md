# Structure and risk patterns

The shape of a program that will survive its second year: layers that hold, seams that are honest, doors that are known, and a blast radius the programmer chose.

When to load: Phase 2 (design), Phase 4 (audit). Role: architect and auditor.

## Layering

- Three layers at most: entry (parse, render), domain (rules), adapter (I/O). A fourth layer needs a reason.
- Dependencies point inward. The domain does not import the adapter, the framework, or the network.
- A layer that exists only to pass calls through is a layer to delete.
- Cross-layer shortcuts are allowed for reads in a small program if named; never for writes.

## Seams

- A seam sits where the program talks to something it does not control: network, clock, filesystem, process, third party.
- A seam is a named interface with one real implementation and one fake. It is not an abstraction layer for its own sake.
- Test the seam's contract, not the third party.
- A seam that leaks the third party's types into the domain is not a seam.

## Coupling

- Two modules that always change together are one module.
- A shared mutable global is a coupling defect, not a convenience.
- A module knows its neighbours' interfaces, not their internals.
- Circular imports are a design finding, not a lint nuisance.

## One-way doors

Name each and record an ADR before the code:

- persisted schema and file formats;
- public API and wire protocol;
- dependency choice with a licence or runtime cost;
- distribution shape (binary, image, package, service);
- auth model and secret storage.

For each: what makes it hard to reverse, what the rollback would cost, and who decided.

## Blast radius

- A failure in this module affects: this request / this process / this host / every user. Write which.
- A migration has a forward and a backward path, tested on a copy.
- A background job is idempotent or it says why not.
- A retry has a bound and a backoff; an unbounded retry is an outage.

## Operability

- One command builds, one runs. Both documented and run from a clean checkout.
- Config precedence is written once and tested: flag over env over file over default.
- Errors a maintainer will see at 3 a.m. name the request, the input and the next step.
- Logs identify the request and the outcome; no secrets.
- Health and readiness are separate where a deploy depends on them.

## Security surface

- Every trust boundary is named, and every input crossing it is validated.
- Secrets come from the environment or a secret store, never the repo.
- The applicable rows of `skillgrid:_shared/references/threat-matrix.md` are closed or explicitly accepted.
