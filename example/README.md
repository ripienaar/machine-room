# Machine Room Example

This is a Docker Compose environment that runs a basic Machine Room agent in a customer node.

See the [ADR](../README.md) for overview and conceptual information.

## Status

It's a early WIP that shows the concept and how we can deliver a turn-key solution for creating
management agents that leans heavily on autonomous agents.

### Todo

 * The `CONFIG` bucket per account is not ideal, better to have one bucket for all customers with a customer having access to just his data, still a few items to resolve
 * The customer accounts in the SaaS should have no JetStream
 * We need to be able to distribute nats credentials from provisioning for replication auth
 * The options.Options should have a list of plugins we want to deny so they do not activate

## Components

### SaaS Side

 * An Apache HTTP server that hosts autonomous agents
 * A Choria Broker and Choria Provisioner that on-boards new customers
 * A NATS JetStream server that mimic the SaaS Backend
   * An account per customer with a `CONFIG` KV Bucket, users for the customer and an admin, customer one is locked down
   * A `backend` account with streams that all have cust id or name in the subjects
     * `MACHINE_ROOM_EVENTS` holds Choria Lifecycle Events and Autonomous Agent events, all Cloud Events format
     * `MACHINE_ROOM_NODES` holds Choria Registration data for every managed node
     * `MACHINE_ROOM_SUBMISSION` holds Choria Submission messages from every managed node

### Customer Side

 * An instance of the `agent/main.go` that's a Machine Room agent
   * Hosts autonomous agents fetched from a web server managed by `plugins` watcher
   * Regularly update and publishes facts
   * Supports Choria Submit to allow autonomous agents to send events back to SaaS
   * Replicates data to and from the SaaS
   * Handles provisioning onto the SaaS system and negotiation credentials and more
   * Provides a local cache of events, metrics etc that will sync to the SaaS should the customer be offline for a while