FROM discoenv/swagger-base

COPY bin/permissions-server /bin/permissions

ARG git_commit=unknown
ARG buildenv_git_commit=unknown
ARG version=unknown
LABEL org.iplantc.de.permissions.git-ref="$git_commit" \
      org.iplantc.de.permissions.version="$version" \
      org.iplantc.de.buildenv.git-ref="$buildenv_git_commit"

ENTRYPOINT ["permissions"]
CMD ["--help"]
