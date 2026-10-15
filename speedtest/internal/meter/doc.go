// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

// Package meter runs the transfer workers of download and upload tests and measures their rates.
//
// A [Meter] holds one download and one upload [Direction]. Request handlers register with a direction, which runs
// them on its workers until the capture time ends, the rate settles, or the test's context ends. Each request reads
// or writes its data through a [Chunk], which counts the bytes. Uploads count bytes once the server confirms them,
// and an adaptive worker count keeps unconfirmed upload data from piling up.
package meter
