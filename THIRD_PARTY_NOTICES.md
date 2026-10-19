# Third-party notices

insideout and insideout-mcp are free software under the Mozilla Public License 2.0
(the file LICENSE in this archive and in the repository). They contain the third-party
components listed below, each under its own license; the texts follow as those licenses
require. E-mail addresses in upstream copyright lines are left out of this file.

## Source code (MPL-2.0 section 3.2)

The Source Code Form of this software is available, at no charge, from the project
repository:

    https://github.com/frankji-groundcontrol/insideout

To get the source of the exact release you hold, check out the tag that `insideout version`
prints (for example `v1.2.3`), or download
`https://github.com/frankji-groundcontrol/insideout/archive/refs/tags/<tag>.tar.gz`.
`go version -m insideout` lists the exact third-party module versions built in. If you
cannot reach the repository, open an issue at
`https://github.com/frankji-groundcontrol/insideout/issues` and ask for the source.

## Components

| Component | License | Copyright |
|---|---|---|
| Go standard library (the toolchain that built the binary) | BSD-3-Clause (Go Authors) | Copyright 2009 The Go Authors. |
| github.com/atotto/clipboard | BSD-3-Clause (Ato Araki) | Copyright (c) 2013 Ato Araki. All rights reserved. |
| github.com/aymanbagabas/go-osc52/v2 | MIT | Copyright (c) 2022 Ayman Bagabas |
| github.com/catppuccin/go | MIT | Copyright (c) 2021 Catppuccin |
| github.com/charmbracelet/bubbles | MIT | Copyright (c) 2020-2023 Charmbracelet, Inc |
| github.com/charmbracelet/bubbletea | MIT | Copyright (c) 2020-2025 Charmbracelet, Inc |
| github.com/charmbracelet/colorprofile | MIT | Copyright (c) 2020-2024 Charmbracelet, Inc |
| github.com/charmbracelet/huh | MIT | Copyright (c) 2023 Charm |
| github.com/charmbracelet/lipgloss | MIT | Copyright (c) 2021-2023 Charmbracelet, Inc |
| github.com/charmbracelet/x/ansi | MIT | Copyright (c) 2023 Charmbracelet, Inc. |
| github.com/charmbracelet/x/cellbuf | MIT | Copyright (c) 2023 Charmbracelet, Inc. |
| github.com/charmbracelet/x/exp/strings | MIT | Copyright (c) 2023 Charmbracelet, Inc. |
| github.com/charmbracelet/x/term | MIT | Copyright (c) 2023 Charmbracelet, Inc. |
| github.com/dustin/go-humanize | MIT (go-humanize text) | Copyright (c) 2005-2008  Dustin Sallings |
| github.com/golang-jwt/jwt/v5 | MIT | Copyright (c) 2012 Dave Grijalva; Copyright (c) 2021 golang-jwt maintainers |
| github.com/google/uuid | BSD-3-Clause (Google) | Copyright (c) 2009,2014 Google Inc. All rights reserved. |
| github.com/jackc/pgpassfile | MIT | Copyright (c) 2019 Jack Christensen |
| github.com/jackc/pgservicefile | MIT | Copyright (c) 2020 Jack Christensen |
| github.com/jackc/pgx/v5 | MIT | Copyright (c) 2013-2021 Jack Christensen |
| github.com/jackc/puddle/v2 | MIT | Copyright (c) 2018 Jack Christensen |
| github.com/lucasb-eyer/go-colorful | MIT | Copyright (c) 2013 Lucas Beyer |
| github.com/mark3labs/mcp-go | MIT | Copyright (c) 2024 Anthropic, PBC |
| github.com/mattn/go-isatty | MIT | Copyright (c) Yasuhiro MATSUMOTO |
| github.com/mattn/go-runewidth | MIT | Copyright (c) 2016 Yasuhiro Matsumoto |
| github.com/mitchellh/hashstructure/v2 | MIT | Copyright (c) 2016 Mitchell Hashimoto |
| github.com/muesli/ansi | MIT | Copyright (c) 2021 Christian Muehlhaeuser |
| github.com/muesli/cancelreader | MIT | Copyright (c) 2022 Erik Geiser and Christian Muehlhaeuser |
| github.com/muesli/termenv | MIT | Copyright (c) 2019 Christian Muehlhaeuser |
| github.com/rivo/uniseg | MIT | Copyright (c) 2019 Oliver Kuederle |
| github.com/spf13/cast | MIT | Copyright (c) 2014 Steve Francia |
| github.com/xo/terminfo | MIT | Copyright (c) 2016 Anmol Sethi |
| github.com/yosida95/uritemplate/v3 | BSD-3-Clause (Kohei Yoshida) | Copyright (C) 2016, Kohei YOSHIDA <https://yosida95.com/>. All rights reserved. |
| golang.org/x/crypto | BSD-3-Clause (Go Authors) | Copyright 2009 The Go Authors. |
| golang.org/x/sync | BSD-3-Clause (Go Authors) | Copyright 2009 The Go Authors. |
| golang.org/x/sys | BSD-3-Clause (Go Authors) | Copyright 2009 The Go Authors. |
| golang.org/x/term | BSD-3-Clause (Go Authors) | Copyright 2009 The Go Authors. |
| golang.org/x/text | BSD-3-Clause (Go Authors) | Copyright 2009 The Go Authors. |
| gopkg.in/yaml.v3 | MIT and Apache-2.0 | see the yaml.v3 text below |

## Fonts in the web client

The Go binaries in this archive contain no fonts. The web client in the
repository serves its own: `client/web/fonts/` holds the 725 font files (145
families, Noto and Roboto) that the Flutter web engine would otherwise fetch from
`fonts.gstatic.com`, unmodified copies in the same layout. They are under the SIL
Open Font License 1.1. The license text of each family is
`client/web/fonts/licenses/<family>.txt`, and `client/web/fonts/NOTICE.txt` lists
the copyright holders: Adobe (the five CJK families), Google (Noto Color Emoji),
The Roboto Project Authors (Roboto) and The Noto Project Authors (the other 138).
The native builds bundle Noto Sans SC and Noto Serif SC under the same license
(`client/assets/fonts/OFL.txt`).

## License texts

### MIT

Applies to every component marked MIT. Each one's copyright line is in the table above;
this permission notice follows it, as the license requires.

```
Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

### MIT (go-humanize text)

```
Copyright (c) 2005-2008  Dustin Sallings

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

<http://www.opensource.org/licenses/mit-license.php>
```

### BSD-3-Clause (Ato Araki)

```
Copyright (c) 2013 Ato Araki. All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are
met:

   * Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.
   * Redistributions in binary form must reproduce the above
copyright notice, this list of conditions and the following disclaimer
in the documentation and/or other materials provided with the
distribution.
   * Neither the name of @atotto. nor the names of its
contributors may be used to endorse or promote products derived from
this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
```

### BSD-3-Clause (Google)

```
Copyright (c) 2009,2014 Google Inc. All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are
met:

   * Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.
   * Redistributions in binary form must reproduce the above
copyright notice, this list of conditions and the following disclaimer
in the documentation and/or other materials provided with the
distribution.
   * Neither the name of Google Inc. nor the names of its
contributors may be used to endorse or promote products derived from
this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
```

### BSD-3-Clause (Kohei Yoshida)

```
Copyright (C) 2016, Kohei YOSHIDA <https://yosida95.com/>. All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

    * Redistributions of source code must retain the above copyright
      notice, this list of conditions and the following disclaimer.
    * Redistributions in binary form must reproduce the above copyright
      notice, this list of conditions and the following disclaimer in the
      documentation and/or other materials provided with the distribution.
    * Neither the name of the copyright holder nor the names of its
      contributors may be used to endorse or promote products derived from
      this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
```

### BSD-3-Clause (Go Authors)

Applies to the Go standard library and golang.org/x/*.

```
Copyright 2009 The Go Authors.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are
met:

   * Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.
   * Redistributions in binary form must reproduce the above
copyright notice, this list of conditions and the following disclaimer
in the documentation and/or other materials provided with the
distribution.
   * Neither the name of Google LLC nor the names of its
contributors may be used to endorse or promote products derived from
this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
```

### gopkg.in/yaml.v3 (MIT and Apache-2.0)

The Apache License 2.0 applies to the parts named below; its full text is at
https://www.apache.org/licenses/LICENSE-2.0.txt

```
This project is covered by two different licenses: MIT and Apache.

#### MIT License ####

The following files were ported to Go from C files of libyaml, and thus
are still covered by their original MIT license, with the additional
copyright staring in 2011 when the project was ported over:

    apic.go emitterc.go parserc.go readerc.go scannerc.go
    writerc.go yamlh.go yamlprivateh.go

Copyright (c) 2006-2010 Kirill Simonov
Copyright (c) 2006-2011 Kirill Simonov

Permission is hereby granted, free of charge, to any person obtaining a copy of
this software and associated documentation files (the "Software"), to deal in
the Software without restriction, including without limitation the rights to
use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies
of the Software, and to permit persons to whom the Software is furnished to do
so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

### Apache License ###

All the remaining project files are covered by the Apache license:

Copyright (c) 2011-2019 Canonical Ltd

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
```

```
Copyright 2011-2016 Canonical Ltd.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
```
