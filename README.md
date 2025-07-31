# Active Sybil Attack and Efficient Defense Strategy in IPFS DHT

This repository contains the source code for the paper [**Active Sybil Attack and Efficient Defense Strategy in IPFS DHT**](https://arxiv.org/abs/2505.01139). It includes implementations of the proposed attack and defense mechanisms described in the paper.

> **Résumé:**
>
> The InterPlanetary File System (IPFS) is a decentralized peer-to-peer (P2P) storage built on Kademlia, a Distributed Hash Table (DHT) structure commonly used in P2P systems and known for its proved scalability. However, DHTs are susceptible to Sybil attacks, where a single entity controls multiple malicious nodes. Recent studies have shown that IPFS is affected by a passive content eclipse attack, leveraging Sybils, in which adversarial nodes hide received indexed information from other peers, making the content appear unavailable. Fortunately, the latest mitigation strategy coupling an attack detection based on statistical tests and a wider publication strategy upon detection was able to circumvent it.
>
> In this work, we present a new active attack in which malicious nodes return semantically correct but intentionally false data. The attack leverages strategic Sybil placement to evade detection mechanisms and exploits an early termination behavior in Kubo, the main implementation of IPFS. Our approach is capable of fully eclipsing content on the real IPFS network. When evaluated against the most recent known mitigation, it successfully denies access to the target content in approximately 80\% of lookup attempts.
>
> To address this vulnerability, we propose a new mitigation called SR-DHT-Store, which enables efficient, Sybil-resistant content publication without relying on attack detection. Instead, it uses systematic and precise use of region-based queries based on a dynamically computed XOR distance to the target ID. SR-DHT-Store can be combined with other defense mechanisms, fully mitigating passive and active Sybil attacks at a lower overhead while supporting an incremental deployment.


## Ethical Considerations

While this paper introduces a new active attack approach targeting IPFS, the recently discovered passive attack remains unmitigated in the current Kubo implementation. Therefore, this experiment does not introduce any new threats beyond those already present in the system. Since the mitigation proposed by Sridhar et al. [[1]](#references) has not yet been integrated into the mainstream client, we do not consider this to be an attack on Kubo itself. 

## Description and Requirements

The tests and experiments must be conducted on a publicly accessible network in order to actively respond to DHT queries. Ports **5001** and **8080** must be open and available for the **Kubo API** and **Gateway**, respectively, when those functionalities are used. Each Sybil node runs on a separate port, which must also be publicly accessible. This port can be specified using the `--port` flag in any of our programs.

Our experiments were performed on a machine located at INRIA Nancy, equipped with an **Intel(R) Core(TM) i7-9700 CPU @ 3.00GHz** and **32 GB of RAM**. The network should be stable, and peers should remain online for several consecutive days to allow for the attackers to populate multiple routing tables.

To compile the project, we recommend using **Go version 1.24.5** or later.

## Project Structure

The structure of our codebase is as follows:

- **[db/](./db/)**: Contains 100 random lookups and the peers contacted along each lookup path. This dataset was used across multiple tests to avoid repeated lookups when performing experiments.
- [**logger/**](./logger/): Provides logging functions that are imported and used throughout all sub-projects.
- [**mitigation/**](./mitigation/): Includes implementations of all proposed mitigation strategies:  
    - **SR-DHT-Store** (provider-side)  
    - [**PR Limitation**](./mitigation/pr-limitation/) (client-side)  
    - [**Disjoint Requests**](./mitigation/disjoint-requests/) (client-side)
- [**node/**](./node/): Defines the types of nodes used in the experiments: regular and Sybil.  
    - Both rely on the node/peer project, which is used to instantiate an IPFS node using kubo-as-library.
- [**tests/**](./tests/): Contains all experiments executed against the IPFS network.
- [**utils/**](./utils/): The core sub-projects imported by `tests/` for performing the experiments.

Each folder contains a `README.md` file with detailed information about its corresponding sub-projects.

## Dependencies

Some projects require modified versions of `go-libp2p-kad-dht` and `go-libp2p`. These can be found at the following paths:

- `go-libp2p-kad-dht`: [https://github.com/vicnetto/go-libp2p-kad-dht](https://github.com/vicnetto/go-libp2p-kad-dht)
- `go-libp2p`: [https://github.com/vicnetto/go-libp2p](https://github.com/vicnetto/go-libp2p)

These repositories are organized into multiple branches, each implementing a specific functionality. Detailed explanations for each branch can be found in the corresponding `README.md` files.

## Installation and Configuration

To begin, clone the repository locally:

```sh
git clone https://github.com/vicnetto/active-sybil-attack-and-efficient-defense-strategy-in-ipfs-dht.git
```

Next, navigate to the desired project folder and download its dependencies:

```sh
cd <project-folder>
go mod tidy
```

To run the binary:

```sh
./<project-folder>
```
> **Note:** For convenience, each project folder shares the same name as its corresponding binary.

When executing a binary, it will display the required flags for proper execution.

## Contact

For further questions, reach out to us via email:
- [Victor Henrique DE MOURA NETTO](mailto:victor-henrique.de-moura-netto@inria.fr)
- [Thibault CHOLEZ](mailto:thibault.cholez@inria.fr)
- [Claudia IGNAT](mailto:cludia.ignat@inria.fr)

## References

[1] S. Sridhar, O. Ascigil, N. Keizer, F. Genon, S. Pierre, Y. Psaras, E. Rivière, M. Król, Content Censorship in the InterPlanetary File System, 2023. URL: [http://arxiv.org/abs/2307.12212](http://arxiv.org/abs/2307.12212). doi: 10.48550/arXiv.2307.12212.