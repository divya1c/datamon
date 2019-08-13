#! /bin/zsh

setopt ERR_EXIT

typeset -a version_tags

version_tags=($(git tag --points-at HEAD |grep '^v'))

if [[ ${#version_tags} -gt 1 ]]; then
    print 'ambiguous tags at HEAD: multiple tags begin with v' 1>&2
    print -- "$version_tags" 1>&2
    exit 1
fi

if [[ ${#version_tags} -eq 1 ]]; then
    release_tag=${version_tags[1]}
else
    hash=$(git show --abbrev-commit |grep '^commit' | cut -d' ' -f 2)
    release_tag="hash-${hash}"
fi

print -- ${release_tag}
