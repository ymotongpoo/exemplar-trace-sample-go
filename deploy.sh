# Copyright 2021 Yoshi Yamaguchi
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

project_id=$(gcloud config get-value project)
export KO_DOCKER_REPO="gcr.io/${project_id}/exemplar-sample"

gcloud beta run deploy exemplar-trace-oc-go \
--project=${project_id} \
--image=$(ko publish .) \
--region=us-west1 \
--no-allow-unauthenticated \
--no-cpu-throttling \
--min-instances=1