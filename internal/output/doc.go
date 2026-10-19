// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

// Package output writes the CLI's server list, version banner, and JSON reports.
//
// [WriteServerList] lists the servers, [JSONReport] and [JSONLReport] write the results as JSON, and their Masked
// variants hide the user's address, coordinates, and distances for --mask-isp. [MaskUser] and [MaskedServerLine]
// mask the lines the progress reporters show.
package output
