# Copyright 2020 Marco Nenciarini <mnencia@gmail.com>
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

TARGET_DIR=${1:-/usr/local/bin}
MCHFUSE_VERSION=$(curl -s -LH "Accept:application/json" "https://github.com/mnencia/mchfuse/releases/latest" | sed 's/.*"tag_name":"\([^"]\+\)".*/\1/')
curl -L -o "${TARGET_DIR}/mchfuse" "https://github.com/mnencia/mchfuse/releases/download/${MCHFUSE_VERSION}/mchfuse-${MCHFUSE_VERSION}-$(uname | tr '[:upper:]' '[:lower:]')-$(uname -m)"
chmod 755 "${TARGET_DIR}/mchfuse"
