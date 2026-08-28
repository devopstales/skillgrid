## ADDED Requirements

### Requirement: Graph visualization page

The system SHALL serve a browser-based graph visualization at `GET /graph` on `skillgraph
serve`, fully offline (embedded assets, no CDN).

#### Scenario: Page and assets served
- **GIVEN** the serve binary is running
- **WHEN** `GET /graph` and its static assets (js/css bundles) are requested
- **THEN** the page and assets return `200` with correct content types from embedded files

#### Scenario: Subgraph rendering
- **GIVEN** graph data exists for a project
- **WHEN** the visualization page loads for that project
- **THEN** nodes are rendered as an interactive force-directed layout with layout, pan, and zoom
- **THEN** nodes are visually distinguished by kind and edges by type

#### Scenario: Node detail on selection
- **GIVEN** the graph is rendered
- **WHEN** a node is selected
- **THEN** the node's source-row content (via existing JSON endpoints) is shown in a detail panel

#### Scenario: Filtering
- **GIVEN** the graph is rendered
- **WHEN** node kind or edge type filters are toggled
- **THEN** the visualization updates to show only the selected subset
