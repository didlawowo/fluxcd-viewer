package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// 🔧 Structure pour les Kustomizations
type Kustomization struct {
	Resource    string      `json:"resource"`
	Namespace   string      `json:"namespace"`
	Path        string      `json:"path"`
	Status      string      `json:"status"`
	LastApplied string      `json:"lastApplied"`
	Conditions  []Condition `json:"conditions"` // Nouveau champ
	Message     string      `json:"message"`    // Message d'erreur principal
	Group       string      `json:"group"`      // Nouveau champ
}

// 🔧 Structure pour les HelmRelease
type HelmRelease struct {
	Resource    string      `json:"resource"`
	Namespace   string      `json:"namespace"`
	ReleaseName string      `json:"releaseName"`
	Interval    string      `json:"interval"`
	Chart       string      `json:"chart"`
	Version     string      `json:"version"`
	Status      string      `json:"status"`
	LastApplied string      `json:"lastApplied"`
	Conditions  []Condition `json:"conditions"` // Nouveau champ
	Message     string      `json:"message"`    // Message d'erreur principal
	Group       string      `json:"group"`      // Nouveau champ
}

type Condition struct {
	Type    string `json:"type"`
	Status  string `json:"status"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

// 🎯 Structure détaillée du Kustomization FluxCD
type FluxKustomization struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Metadata   struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	} `json:"metadata"`
	Spec struct {
		Path      string `json:"path"`
		Prune     bool   `json:"prune"`
		SourceRef struct {
			Kind string `json:"kind"`
			Name string `json:"name"`
		} `json:"sourceRef"`
	} `json:"spec"`
	Status struct {
		LastAppliedRevision string `json:"lastAppliedRevision"`
		Conditions          []Condition
	} `json:"status"`
}

// 🎯 Structure détaillée d'une HelmRelease FluxCD
type FluxHelmRelease struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Metadata   struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	} `json:"metadata"`
	Spec struct {
		ReleaseName string `json:"releaseName"`
		Interval    string `json:"interval"`
		Chart       struct {
			Spec struct {
				Chart   string `json:"chart"`
				Version string `json:"version"`
			} `json:"spec"`
		} `json:"chart"`
	} `json:"spec"`
	Status struct {
		LastAppliedRevision string `json:"lastAppliedRevision"`
		Conditions          []Condition
	} `json:"status"`
}

func getKustomizations() ([]Kustomization, error) {
	config, err := getK8sClient()
	if err != nil {
		return nil, fmt.Errorf("❌ erreur connexion cluster: %v", err)
	}

	// 🚀 Création du client dynamique
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	// 📦 Définition de la ressource Kustomization de FluxCD
	gvr := schema.GroupVersionResource{
		Group:    "kustomize.toolkit.fluxcd.io", // Groupe API FluxCD
		Version:  "v1",                          // Version de l'API
		Resource: "kustomizations",              // Type de ressource
	}

	// 📥 Récupération des Kustomizations
	results, err := dynamicClient.Resource(gvr).Namespace("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Printf("❌ Erreur lors de la récupération des kustomizations: %v", err)
		return nil, err
	}

	var kustomizations []Kustomization

	// 🔄 Traitement des résultats
	for _, result := range results.Items {
		resultData, err := result.MarshalJSON()
		if err != nil {
			log.Printf("❌ Erreur marshaling: %v", err)
			continue
		}

		var fluxKusto FluxKustomization
		if err := json.Unmarshal(resultData, &fluxKusto); err != nil {
			log.Printf("❌ Erreur parsing: %v", err)
			continue
		}

		// Extraction des conditions et du message d'erreur
		var conditions []Condition
		mainMessage := ""

		for _, cond := range fluxKusto.Status.Conditions {
			condition := Condition{
				Type:    cond.Type,
				Status:  cond.Status,
				Reason:  cond.Reason,
				Message: cond.Message,
			}
			conditions = append(conditions, condition)

			// Si la condition est Ready et False, on garde le message d'erreur
			if cond.Type == "Ready" && cond.Status == "False" {
				mainMessage = cond.Message
			}
		}

		// Construction de l'objet Kustomization
		kusto := Kustomization{
			Resource:    fluxKusto.Kind + "/" + fluxKusto.Metadata.Name,
			Namespace:   fluxKusto.Metadata.Namespace,
			Path:        fluxKusto.Spec.Path,
			Status:      getStatusFromConditions(conditions),
			LastApplied: fluxKusto.Status.LastAppliedRevision,
			Conditions:  conditions,
			Message:     mainMessage,
			Group:       extractGroupFromPath(fluxKusto.Spec.Path),
		}

		kustomizations = append(kustomizations, kusto)
	}

	return kustomizations, nil
}

func getHelmReleases() ([]HelmRelease, error) {
	config, err := getK8sClient()
	if err != nil {
		return nil, fmt.Errorf("❌ erreur connexion cluster: %v", err)
	}

	// 🚀 Création du client dynamique
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	// 📦 Définition de la ressource HelmRelease de FluxCD
	gvr := schema.GroupVersionResource{
		Group:    "helm.toolkit.fluxcd.io", // Groupe API FluxCD
		Version:  "v2",                     // Version de l'API
		Resource: "helmreleases",           // Type de ressource
	}

	// 📥 Récupération des HelmReleases
	results, err := dynamicClient.Resource(gvr).Namespace("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Printf("❌ Erreur lors de la récupération des HelmReleases: %v", err)
		return nil, err
	}

	var helmReleases []HelmRelease

	// 🔄 Traitement des résultats
	for _, result := range results.Items {
		resultData, err := result.MarshalJSON()
		if err != nil {
			log.Printf("❌ Erreur marshaling: %v", err)
			continue
		}

		var fluxHelm FluxHelmRelease
		if err := json.Unmarshal(resultData, &fluxHelm); err != nil {
			log.Printf("❌ Erreur parsing: %v", err)
			continue
		}

		// Extraction des conditions et du message d'erreur
		var conditions []Condition
		mainMessage := ""

		for _, cond := range fluxHelm.Status.Conditions {
			condition := Condition{
				Type:    cond.Type,
				Status:  cond.Status,
				Reason:  cond.Reason,
				Message: cond.Message,
			}
			conditions = append(conditions, condition)

			// Si la condition est Ready et False, on garde le message d'erreur
			if cond.Type == "Ready" && cond.Status == "False" {
				mainMessage = cond.Message
			}
		}

		// Construction de l'objet HelmRelease
		helm := HelmRelease{
			Resource:    fluxHelm.Kind + "/" + fluxHelm.Metadata.Name,
			Namespace:   fluxHelm.Metadata.Namespace,
			ReleaseName: fluxHelm.Spec.ReleaseName,
			Interval:    fluxHelm.Spec.Interval,
			Chart:       fluxHelm.Spec.Chart.Spec.Chart,
			Version:     fluxHelm.Spec.Chart.Spec.Version,
			Status:      getStatusFromConditions(conditions),
			LastApplied: fluxHelm.Status.LastAppliedRevision,
			Conditions:  conditions,
			Message:     mainMessage,
			Group:       extractGroupFromPath(fluxHelm.Spec.Chart.Spec.Chart),
		}

		helmReleases = append(helmReleases, helm)
	}

	return helmReleases, nil
}

func getStatusFromConditions(conditions []Condition) string {
	for _, cond := range conditions {
		if cond.Type == "Ready" {
			return cond.Status
		}
	}
	return "Unknown"
}

func getK8sClient() (*rest.Config, error) {
	// 🔍 Tente d'abord de charger la configuration in-cluster
	config, err := rest.InClusterConfig()
	if err != nil {
		// 🏠 Si échec, essaie la config locale
		home := homedir.HomeDir()
		kubeconfig := filepath.Join(home, ".kube", "config")

		// 🔧 Construit la config depuis le fichier kubeconfig
		config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("❌ impossible de charger la configuration Kubernetes: %v", err)
		}
		return config, nil
	}
	return config, nil
}

func extractGroupFromPath(path string) string {
	// Divise le chemin en segments
	segments := strings.Split(path, "/")

	// Nous voulons ignorer des segments vides ou génériques
	for _, segment := range segments {
		// Ignore les segments vides ou le point
		if segment == "" || segment == "." {
			continue
		}

		// Retourne le premier segment significatif
		return segment
	}

	// Si aucun segment valide n'est trouvé, retourne "other"
	return "other"
}

func main() {
	// 📝 Log de démarrage
	log.Printf("🚀 Démarrage du serveur Frontend FluxCD Kustomizations...")

	// Servir les fichiers statiques
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Routes
	http.HandleFunc("/", handleHome)
	http.HandleFunc("/analyze", handleAnalyze)
	http.HandleFunc("/details", handleDetails)

	// ✨ Nouvelle route health check
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		// Vérifie la connexion au cluster
		_, err := getK8sClient()
		if err != nil {
			log.Printf("❌ Health check échoué: %v", err)
			http.Error(w, "Kubernetes connection failed", http.StatusServiceUnavailable)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// 📢 Logs plus détaillés
	log.Printf("🛣️  Routes configurées: /, /analyze, /details, /health")
	log.Printf("🌐 Serveur prêt sur http://localhost:8080")

	// Démarrage du serveur
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("❌ Erreur démarrage serveur: %v", err)
	}
}

func handleDetails(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	namespace := r.URL.Query().Get("namespace")

	kustomizations, err := getKustomizations()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	helmreleases, err := getHelmReleases()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Recherche de la kustomization spécifique
	for _, ks := range kustomizations {
		if ks.Resource == name && ks.Namespace == namespace {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(ks)
			return
		}
	}

	// Recherche de la helmrelease spécifique
	for _, hr := range helmreleases {
		if hr.Resource == name && hr.Namespace == namespace {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(hr)
			return
		}
	}

	http.Error(w, "Resource not found", http.StatusNotFound)
}
func handleAnalyze(w http.ResponseWriter, r *http.Request) {
	kustomizations, err := getKustomizations()
	if err != nil {
		log.Printf("❌ Erreur analyse: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(kustomizations); err != nil {
		log.Printf("❌ Erreur encodage JSON: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	// Création des fonctions template
	funcMap := template.FuncMap{
		"getCategoryFromPath": func(path string) string {
			if strings.Contains(path, "/apis/") {
				return "apis"
			} else if strings.Contains(path, "/addons/") {
				return "addons"
			} else if strings.Contains(path, "/apps/") {
				return "apps"
			}
			return "other"
		},
		"shortenCommit": func(hash string) string {
			if len(hash) > 7 {
				return hash[:7]
			}
			return hash
		},
		"getBranch": func(lastApplied string) string {
			parts := strings.Split(lastApplied, "/")
			if len(parts) > 0 {
				return parts[0] // Retourne la partie avant le '/'
			}
			return lastApplied
		},
		"countHealthy": func(kustomizations []Kustomization) int {
			count := 0
			for _, k := range kustomizations {
				if k.Status == "True" {
					count++
				}
			}
			return count
		},
		"countFailed": func(kustomizations []Kustomization) int {
			count := 0
			for _, k := range kustomizations {
				if k.Status != "True" {
					count++
				}
			}
			return count
		},
		"getUniqueGroups": func(kustomizations []Kustomization, helmreleases []HelmRelease) map[string]bool {
			groups := make(map[string]bool)
			for _, ks := range kustomizations {
				groups[ks.Group] = true
			}
			for _, hr := range helmreleases {
				groups[hr.Group] = true
			}
			return groups
		},
		"title": strings.Title,
	}

	// Parse le template avec les fonctions
	tmpl := template.Must(template.New("index.html").Funcs(funcMap).ParseFiles("views/index.html"))

	analysesks, err := getKustomizations()
	if err != nil {
		log.Printf("Erreur lors de l'analyse: %v", err)
		http.Error(w, "Erreur lors de l'analyse", http.StatusInternalServerError)
		return
	}

	analyseshr, err := getHelmReleases()
	if err != nil {
		log.Printf("Erreur lors de l'analyse: %v", err)
		http.Error(w, "Erreur lors de l'analyse", http.StatusInternalServerError)
		return
	}

	data := struct {
		Kustomizations []Kustomization
		HelmReleases   []HelmRelease
	}{
		Kustomizations: analysesks,
		HelmReleases:   analyseshr,
	}

	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Erreur template: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
