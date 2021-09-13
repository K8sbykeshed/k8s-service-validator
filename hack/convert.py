import re
import requests
from requests.auth import HTTPBasicAuth

from nltk.stem.wordnet import WordNetLemmatizer
import re, string, unicodedata
from nltk.corpus import stopwords
from nltk.tokenize import word_tokenize, sent_tokenize
from nltk.stem.wordnet import WordNetLemmatizer

# Github settings
repo = ""
username = ""
token = ""

auth = HTTPBasicAuth(username, token)
path = "/home/knabben/go/src/k8s.io/kubernetes/test/e2e/network/service.go"  # any file

lemmatizer = WordNetLemmatizer()


def remove_non_ascii(words):
    """Remove non-ASCII characters from list of tokenized words"""
    new_words = []
    for word in words:
        new_word = (
            unicodedata.normalize("NFKD", word)
            .encode("ascii", "ignore")
            .decode("utf-8", "ignore")
        )
        new_words.append(new_word)
    return new_words


def to_lowercase(words):
    """Convert all characters to lowercase from list of tokenized words"""
    new_words = []
    for word in words:
        new_word = word.lower()
        new_words.append(new_word)
    return new_words


def remove_punctuation(words):
    """Remove punctuation from list of tokenized words"""
    new_words = []
    for word in words:
        new_word = re.sub(r"[^\w\s]", "", word)
        if new_word != "":
            new_words.append(new_word)
    return new_words


def remove_stopwords(words):
    """Remove stop words from list of tokenized words"""
    new_words = []
    for word in words:
        if word not in stopwords:
            new_words.append(word)
    return new_words


def lemmatize_list(words):
    new_words = []
    for word in words:
        new_words.append(lemmatizer.lemmatize(word, pos="v"))
    return new_words


def normalize(words):
    words = remove_non_ascii(words)
    words = to_lowercase(words)
    words = remove_punctuation(words)
    words = lemmatize_list(words)
    return " ".join(words)


def clean_var(text):
    data = re.search('"(.*)"', text)[1]
    return normalize(nltk.word_tokenize(data))  # Tokenization of data


items = {}
title = None

# cleanup and convert items
for idx, n in enumerate(open(path, "r").readlines()):
    if "ginkgo.By" in n:
        item = clean_var(n)
        if title:
            items[title] += f"{item}\n"
    if "ginkgo.It" in n:
        item = clean_var(n)
        title = item
        if title not in items:
            items[title] = ""


def create_issue(title, body):
    payload = {"title": title, "body": body}
    requests.post(f"{repo}/issues", json=payload, auth=auth)


# create the issues in the repository
for k, v in items.items():
    create_issue(k.title(), v)
