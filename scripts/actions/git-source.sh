# git-source - stand up the lightest viable in-cluster git server (smart
# HTTP), and push the compiled $OUT tree to it. Harness scaffolding, not a
# declared environment prerequisite and not compiled output: it exists
# only for this run's lifetime, same as the kind cluster itself. The
# `deliver` action registers this content with Flux (the GitRepository
# CR) and reconciles it - kept separate so this action's job stays just
# "the git source exists and holds $OUT".
require_repo_root

log "git source: stand up the lightest viable in-cluster git server (smart HTTP)"
GITSERVER_BUILD_DIR="$(mktemp -d)"
cat >"$GITSERVER_BUILD_DIR/Dockerfile" <<'DOCKERFILE'
# Harness-only git server: not compiled output, not product code. It
# exists solely to give Flux's source-controller a real git source to
# clone from, over the git smart-HTTP protocol (the only transports
# Flux's GitRepository CRD accepts are http/https/ssh - no git:// or
# file://, checked against the upstream CRD schema before this was
# written).
FROM debian:12-slim
RUN apt-get update -qq \
	&& DEBIAN_FRONTEND=noninteractive apt-get install -y -qq --no-install-recommends \
	   git nginx-light fcgiwrap ca-certificates \
	&& rm -rf /var/lib/apt/lists/* \
	&& mkdir -p /srv/git
COPY nginx.conf /etc/nginx/nginx.conf
EXPOSE 80
CMD ["sh", "-c", "fcgiwrap -s unix:/run/fcgiwrap.sock & exec nginx -g 'daemon off;'"]
DOCKERFILE
cat >"$GITSERVER_BUILD_DIR/nginx.conf" <<'NGINXCONF'
user root;
worker_processes 1;
pid /run/nginx.pid;
events { worker_connections 64; }
http {
	server {
		listen 80 default_server;
		location ~ ^/d7s\.git(/.*)?$ {
			root /srv/git;
			client_max_body_size 0;
			include fastcgi_params;
			fastcgi_param SCRIPT_FILENAME /usr/lib/git-core/git-http-backend;
			fastcgi_param GIT_HTTP_EXPORT_ALL "";
			fastcgi_param GIT_PROJECT_ROOT /srv/git;
			fastcgi_param PATH_INFO $uri;
			fastcgi_pass unix:/run/fcgiwrap.sock;
		}
	}
}
NGINXCONF
docker build -q -t "$GITSERVER_IMAGE" "$GITSERVER_BUILD_DIR" >/dev/null
kind load docker-image "$GITSERVER_IMAGE" --name "$CLUSTER_NAME"
docker rmi "$GITSERVER_IMAGE" >/dev/null 2>&1 || true
rm -rf "$GITSERVER_BUILD_DIR"

kubectl create namespace "$GITSERVER_NS" --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: d7s-gitserver
  namespace: $GITSERVER_NS
spec:
  replicas: 1
  selector:
    matchLabels: {app: d7s-gitserver}
  template:
    metadata:
      labels: {app: d7s-gitserver}
    spec:
      containers:
        - name: gitserver
          image: $GITSERVER_IMAGE
          imagePullPolicy: IfNotPresent
          ports:
            - containerPort: 80
---
apiVersion: v1
kind: Service
metadata:
  name: d7s-gitserver
  namespace: $GITSERVER_NS
spec:
  selector: {app: d7s-gitserver}
  ports:
    - port: 80
      targetPort: 80
EOF
kubectl wait --for=condition=Available deployment/d7s-gitserver -n "$GITSERVER_NS" --timeout="$TIMEOUT"
GITSERVER_POD=$(kubectl get pod -n "$GITSERVER_NS" -l app=d7s-gitserver -o jsonpath='{.items[0].metadata.name}')

log "git source: git add out/ && git commit  # the plan is the git diff"
GITCONTENT_DIR="$(mktemp -d)"
mkdir -p "$GITCONTENT_DIR/repo/out"
cp -r "$OUT"/. "$GITCONTENT_DIR/repo/out/"
git -C "$GITCONTENT_DIR/repo" -c init.defaultBranch=main init -q
git -C "$GITCONTENT_DIR/repo" add -A
# commit.gpgsign=false is scoped to ONLY this throwaway scratch repo (a
# -c override on this one invocation, never the developer's global git
# config): it is harness scaffolding pushed to an in-cluster git server
# for Flux to clone, never project history, so it needs no signature -
# and requiring one made every run depend on the developer's gpg-agent
# for something that isn't the developer's commit at all (found live,
# 2026-07-26: a stale gpg-agent cache blocked this step with no bearing
# on the actual product commit).
git -C "$GITCONTENT_DIR/repo" -c user.email=harness@d7s.dev -c user.name=d7s-harness \
	-c commit.gpgsign=false \
	commit -q -m "compiled output (acceptance harness run)"
# kubectl cp's tar-based extraction lands at
# <dest-dirname>/<src-basename> - naming the local bare clone "d7s.git"
# to match the target path exactly (no trailing-slash "copy contents"
# ambiguity to get wrong).
git clone --bare -q "$GITCONTENT_DIR/repo" "$GITCONTENT_DIR/d7s.git"
# Clean-slate the served repo before every publish: each publish is a
# FRESH, unrelated history (git init above), and tar-overlaying a new
# bare repo onto an old one (kubectl cp merges, never replaces) left the
# old packed-refs/objects mixed with the new — source-controller kept
# fetching the pre-publish revision (found live, 2026-07-27: the pin
# ceremony's republished revision never produced a NewArtifact; CI run
# 30301385471).
kubectl exec -n "$GITSERVER_NS" "$GITSERVER_POD" -- rm -rf /srv/git/d7s.git
kubectl cp "$GITCONTENT_DIR/d7s.git" "$GITSERVER_NS/$GITSERVER_POD:/srv/git/d7s.git"
rm -rf "$GITCONTENT_DIR"
